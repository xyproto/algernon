package revoke

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/crypto/ocsp"
)

const (
	// DefaultTimeout is the time allowed for each individual CRL, issuer certificate, or OCSP request when no other
	// timeout is configured.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxCRLSize is the largest CRL response body accepted when no other limit is configured.
	DefaultMaxCRLSize int64 = 16 << 20

	// DefaultMaxResponseSize is the largest issuer certificate or OCSP response body accepted when no other limit is
	// configured.
	DefaultMaxResponseSize int64 = 1 << 20

	// DefaultMaxOCSPAge is the greatest age accepted for an OCSP response which does not state when it will next be
	// updated when no other limit is configured. It is the longest validity interval the CA/Browser Forum Baseline
	// Requirements permit for an OCSP response.
	DefaultMaxOCSPAge = 10 * 24 * time.Hour
)

var (
	// HTTPClient is an instance of http.Client that will be used for all HTTP requests made by the package level
	// functions, and by a Verifier which does not have its own. Every request is additionally bound by a timeout
	// applied via the request context, so a client without a timeout such as http.DefaultClient cannot hang a check.
	HTTPClient = http.DefaultClient

	// HardFail determines whether the failure to check the revocation status of a certificate (i.e. due to network
	// failure or revocation data which can't be authenticated) causes the package level functions to report the
	// certificate as revoked (a hard failure).
	HardFail = false

	crlRead    = io.ReadAll
	remoteRead = io.ReadAll
	ocspRead   = io.ReadAll

	ocspOpts = ocsp.RequestOptions{
		Hash: crypto.SHA1,
	}

	defaultCRLs = &crlCache{}
)

// Verifier checks the revocation status of certificates against their issuer. The zero value is ready to use, and a
// Verifier is safe for concurrent use provided its fields are not modified after first use. Each Verifier has its own
// CRL cache.
type Verifier struct {
	// HTTPClient is used for all requests. When nil the package level HTTPClient is used.
	HTTPClient *http.Client

	// HardFail reports a certificate whose revocation status can't be determined as revoked. The ok result is false in
	// that case regardless of this value.
	HardFail bool

	// Timeout bounds each individual CRL or OCSP request, including reading the response body. Zero means
	// DefaultTimeout, and a negative value disables the timeout so that only the context and HTTPClient apply.
	Timeout time.Duration

	// MaxCRLSize is the largest CRL response body accepted in bytes. Zero or less means DefaultMaxCRLSize.
	MaxCRLSize int64

	// MaxResponseSize is the largest OCSP response body accepted in bytes. Zero or less means DefaultMaxResponseSize.
	MaxResponseSize int64

	// MaxOCSPAge is the greatest age accepted for an OCSP response which does not state when it will next be updated.
	// Zero or less means DefaultMaxOCSPAge. A response which does state when it will next be updated is used until
	// that time regardless of this value.
	MaxOCSPAge time.Duration

	crls crlCache
}

// VerifyCertificate checks that cert is within its validity period, and then checks its revocation status using
// revocation data which must be authenticated by issuer. The issuer must be the certificate which issued cert taken
// from a chain the caller has already verified, for example the next element of a chain returned by
// x509.Certificate.Verify. This is confirmed by checking the signature of cert against issuer, but no trust decision
// about issuer itself is made.
//
// The results are:
//
//	revoked=false, ok=true:  the status was determined and the certificate is not revoked.
//	revoked=true,  ok=true:  the status was determined and the certificate is revoked, or the certificate is outside
//	                         its validity period in which case err wraps ErrCertificateExpired or
//	                         ErrCertificateNotYetValid.
//	revoked=false, ok=false: the status can't be determined and HardFail is false.
//	revoked=true,  ok=false: the status can't be determined and HardFail is true.
//
// Whenever ok is false err describes why the status can't be determined.
//
// Every HTTP or HTTPS CRL distribution point is checked in order. The certificate is revoked as soon as the CRL of any
// of them lists it, without the remaining ones being checked, and is only not revoked once all of them have been
// checked, none of them list it, and at least one of them can be used. When none of the CRLs can be used, or there is
// no supported CRL distribution point, each HTTP or HTTPS OCSP server is tried in order until one of them gives an
// authenticated answer. A CRL is only used when it is signed by issuer, names issuer as its issuer, is
// current and states when it will next be updated, is not a delta or indirect CRL, has no critical extension which is
// not supported, and has a scope which covers cert. An OCSP response is only used when it identifies cert as issued
// by issuer, is current, and is signed by issuer or by a responder certificate which issuer directly issued for OCSP
// signing. A CRL or OCSP response which fails any of these checks never results in a not revoked answer. A CRL which
// is limited to some revocation reasons is only used to report a certificate it lists as revoked.
//
// A certificate which names neither a CRL distribution point nor an OCSP server has no revocation status to check and
// is reported as not revoked with ok=true, even when HardFail is true. A certificate which only names sources that
// are not supported, such as LDAP distribution points, has a status which can't be determined.
func (v *Verifier) VerifyCertificate(ctx context.Context, cert, issuer *x509.Certificate) (revoked, ok bool, err error) {
	client := v.HTTPClient
	if client == nil {
		client = HTTPClient
	}

	c := newChecker(client, v.HardFail, &v.crls)

	if v.Timeout != 0 {
		c.timeout = v.Timeout
	}

	if v.MaxCRLSize > 0 {
		c.maxCRL = v.MaxCRLSize
	}

	if v.MaxResponseSize > 0 {
		c.maxResponse = v.MaxResponseSize
	}

	if v.MaxOCSPAge > 0 {
		c.maxOCSPAge = v.MaxOCSPAge
	}

	return c.check(ctx, cert, staticIssuer(issuer))
}

// VerifyCertificateWithIssuer checks that cert is within its validity period, and then checks its revocation status
// using revocation data which must be authenticated by issuer. It behaves as Verifier.VerifyCertificate does, using
// the package level HTTPClient and HardFail values, the default timeout and size limits, and a shared CRL cache.
func VerifyCertificateWithIssuer(ctx context.Context, cert, issuer *x509.Certificate) (revoked, ok bool, err error) {
	return newChecker(HTTPClient, HardFail, defaultCRLs).check(ctx, cert, staticIssuer(issuer))
}

// VerifyCertificate ensures that the certificate passed in hasn't expired and checks its revocation status. See
// VerifyCertificateError for the details.
//
// Deprecated: use VerifyCertificateWithIssuer or a Verifier with the issuer from a verified certificate chain.
func VerifyCertificate(cert *x509.Certificate) (revoked, ok bool) {
	revoked, ok, _ = VerifyCertificateError(cert)

	return revoked, ok
}

// VerifyCertificateError ensures that the certificate passed in hasn't expired and checks its revocation status. The
// results are the same as those of VerifyCertificateWithIssuer.
//
// As the issuer is not provided it is downloaded from the issuing certificate URLs of the certificate, and is only
// used when its subject matches the issuer of the certificate and it verifies the signature of the certificate. When no
// such issuer can be obtained the revocation status can't be determined. A self-signed certificate is its own issuer.
// This binds the revocation data to the key which signed the certificate rather than to a verified chain, so the
// certificate itself must already be trusted by the caller.
//
// Deprecated: use VerifyCertificateWithIssuer or a Verifier with the issuer from a verified certificate chain.
func VerifyCertificateError(cert *x509.Certificate) (revoked, ok bool, err error) {
	c := newChecker(HTTPClient, HardFail, defaultCRLs)

	return c.check(context.Background(), cert, c.remoteIssuer)
}

// SetCRLFetcher sets the function to use to read from the http response body. The reader it is given is limited to
// one byte more than the applicable size limit.
func SetCRLFetcher(fn func(io.Reader) ([]byte, error)) {
	crlRead = fn
}

// SetRemoteFetcher sets the function to use to read from the http response body. The reader it is given is limited to
// one byte more than the applicable size limit.
func SetRemoteFetcher(fn func(io.Reader) ([]byte, error)) {
	remoteRead = fn
}

// SetOCSPFetcher sets the function to use to read from the http response body. The reader it is given is limited to
// one byte more than the applicable size limit.
func SetOCSPFetcher(fn func(io.Reader) ([]byte, error)) {
	ocspRead = fn
}

type issuerFunc func(ctx context.Context, cert *x509.Certificate) (issuer *x509.Certificate, err error)

type checker struct {
	client      *http.Client
	hardFail    bool
	timeout     time.Duration
	maxCRL      int64
	maxResponse int64
	maxOCSPAge  time.Duration
	crls        *crlCache
}

func newChecker(client *http.Client, hardFail bool, crls *crlCache) *checker {
	if client == nil {
		client = http.DefaultClient
	}

	return &checker{
		client:      client,
		hardFail:    hardFail,
		timeout:     DefaultTimeout,
		maxCRL:      DefaultMaxCRLSize,
		maxResponse: DefaultMaxResponseSize,
		maxOCSPAge:  DefaultMaxOCSPAge,
		crls:        crls,
	}
}

func staticIssuer(issuer *x509.Certificate) issuerFunc {
	return func(_ context.Context, _ *x509.Certificate) (*x509.Certificate, error) {
		if issuer == nil {
			return nil, ErrNoIssuer
		}

		return issuer, nil
	}
}

func (c *checker) undetermined(err error) (revoked, ok bool, e error) {
	return c.hardFail, false, err
}

func (c *checker) check(ctx context.Context, cert *x509.Certificate, getIssuer issuerFunc) (revoked, ok bool, err error) {
	if cert == nil {
		return c.undetermined(errors.New("certificate is nil"))
	}

	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()

	if !now.Before(cert.NotAfter) {
		return true, true, fmt.Errorf("%w at %s", ErrCertificateExpired, cert.NotAfter.Format(time.RFC3339))
	} else if !now.After(cert.NotBefore) {
		return true, true, fmt.Errorf("%w until %s", ErrCertificateNotYetValid, cert.NotBefore.Format(time.RFC3339))
	}

	crls, servers := supportedURLs(cert.CRLDistributionPoints), supportedURLs(cert.OCSPServer)

	if len(crls) == 0 && len(servers) == 0 {
		if len(cert.CRLDistributionPoints) == 0 && len(cert.OCSPServer) == 0 {
			return false, true, nil
		}

		return c.undetermined(ErrNoSupportedSource)
	}

	var issuer *x509.Certificate

	if issuer, err = getIssuer(ctx, cert); err != nil {
		return c.undetermined(err)
	}

	if err = checkIssuer(cert, issuer); err != nil {
		return c.undetermined(err)
	}

	var (
		errs     []error
		complete bool
	)

	for _, uri := range crls {
		if revoked, err = c.checkCRL(ctx, cert, issuer, uri); err != nil {
			errs = append(errs, err)

			continue
		}

		if revoked {
			return true, true, nil
		}

		complete = true
	}

	if complete {
		return false, true, nil
	}

	if len(servers) != 0 {
		var req []byte

		if req, err = ocsp.CreateRequest(cert, issuer, &ocspOpts); err != nil {
			return c.undetermined(errors.Join(append(errs, fmt.Errorf("failed to create OCSP request: %w", err))...))
		}

		for _, server := range servers {
			if revoked, err = c.checkOCSP(ctx, cert, issuer, server, req); err == nil {
				return revoked, true, nil
			}

			errs = append(errs, err)
		}
	}

	return c.undetermined(errors.Join(errs...))
}

func (c *checker) remoteIssuer(ctx context.Context, cert *x509.Certificate) (issuer *x509.Certificate, err error) {
	if checkIssuer(cert, cert) == nil {
		return cert, nil
	}

	errs := []error{ErrNoIssuer}

	for _, uri := range supportedURLs(cert.IssuingCertificateURL) {
		if issuer, err = c.fetchCertificate(ctx, uri); err == nil {
			if err = checkIssuer(cert, issuer); err == nil {
				return issuer, nil
			}
		}

		errs = append(errs, err)
	}

	return nil, errors.Join(errs...)
}

func (c *checker) fetchCertificate(ctx context.Context, uri string) (cert *x509.Certificate, err error) {
	var data []byte

	if data, err = c.fetch(ctx, http.MethodGet, uri, "", nil, remoteRead, c.maxResponse); err != nil {
		return nil, fmt.Errorf("failed to retrieve issuer certificate: %w", err)
	}

	if p, _ := pem.Decode(data); p != nil {
		cert, err = ParseCertificatePEM(data)
	} else {
		cert, err = x509.ParseCertificate(data)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse issuer certificate from %q: %w", uri, err)
	}

	return cert, nil
}

func (c *checker) fetch(ctx context.Context, method, uri, contentType string, body []byte, read func(io.Reader) ([]byte, error), limit int64) (data []byte, err error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.timeout)

		defer cancel()
	}

	var (
		reader io.Reader
		req    *http.Request
		resp   *http.Response
	)

	if body != nil {
		reader = bytes.NewReader(body)
	}

	if req, err = http.NewRequestWithContext(ctx, method, uri, reader); err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if resp, err = c.client.Do(req); err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %d from %q", resp.StatusCode, uri)
	}

	if resp.ContentLength > limit {
		return nil, fmt.Errorf("%w: %q declares %d bytes with a limit of %d", ErrResponseTooLarge, uri, resp.ContentLength, limit)
	}

	if data, err = read(io.LimitReader(resp.Body, limit+1)); err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", uri, err)
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %q exceeds the limit of %d bytes", ErrResponseTooLarge, uri, limit)
	}

	return data, nil
}

func checkIssuer(cert, issuer *x509.Certificate) (err error) {
	if issuer == nil {
		return ErrNoIssuer
	}

	if !bytes.Equal(cert.RawIssuer, issuer.RawSubject) {
		return fmt.Errorf("%w: the certificate issuer name does not match the issuer subject name", ErrIssuerMismatch)
	}

	if err = cert.CheckSignatureFrom(issuer); err != nil {
		return fmt.Errorf("%w: %w", ErrIssuerMismatch, err)
	}

	return nil
}
