package revoke

import (
	"bytes"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/crypto/ocsp"
)

func (c *checker) checkOCSP(ctx context.Context, cert, issuer *x509.Certificate, server string, req []byte) (revoked bool, err error) {
	var data []byte

	if len(req) > 256 {
		data, err = c.fetch(ctx, http.MethodPost, server, "application/ocsp-request", req, ocspRead, c.maxResponse)
	} else {
		data, err = c.fetch(ctx, http.MethodGet, strings.TrimSuffix(server, "/")+"/"+url.QueryEscape(base64.StdEncoding.EncodeToString(req)), "", nil, ocspRead, c.maxResponse)
	}

	if err != nil {
		return false, fmt.Errorf("failed to retrieve OCSP response: %w", err)
	}

	var resp *ocsp.Response

	if resp, err = ocsp.ParseResponseForCert(data, cert, nil); err != nil {
		return false, fmt.Errorf("%w from %q: %w", ErrOCSPInvalid, server, err)
	}

	if err = verifyOCSP(resp, cert, issuer, time.Now(), c.maxOCSPAge); err != nil {
		return false, fmt.Errorf("%w from %q: %w", ErrOCSPInvalid, server, err)
	}

	switch resp.Status {
	case ocsp.Good:
		return false, nil
	case ocsp.Revoked:
		return true, nil
	default:
		return false, fmt.Errorf("%w from %q: the responder does not know the certificate", ErrOCSPInvalid, server)
	}
}

func verifyOCSP(resp *ocsp.Response, cert, issuer *x509.Certificate, now time.Time, maxAge time.Duration) (err error) {
	if responder := resp.Certificate; responder == nil || bytes.Equal(responder.Raw, issuer.Raw) {
		if err = resp.CheckSignatureFrom(issuer); err != nil {
			return fmt.Errorf("the response was not signed by the certificate issuer: %w", err)
		}
	} else {
		switch {
		case !bytes.Equal(responder.RawIssuer, issuer.RawSubject):
			return errors.New("the responder certificate was not issued by the certificate issuer")
		case !slices.Contains(responder.ExtKeyUsage, x509.ExtKeyUsageOCSPSigning):
			return errors.New("the responder certificate is not authorized for OCSP signing")
		case now.Before(responder.NotBefore) || now.After(responder.NotAfter):
			return errors.New("the responder certificate is not within its validity period")
		}

		if err = responder.CheckSignatureFrom(issuer); err != nil {
			return fmt.Errorf("the responder certificate was not issued by the certificate issuer: %w", err)
		}

		if err = resp.CheckSignatureFrom(responder); err != nil {
			return fmt.Errorf("the response was not signed by the responder certificate: %w", err)
		}
	}

	if err = verifyOCSPCertID(resp, cert, issuer); err != nil {
		return err
	}

	if resp.ThisUpdate.After(now.Add(clockSkew)) {
		return fmt.Errorf("not valid until %s", resp.ThisUpdate.Format(time.RFC3339))
	}

	if resp.NextUpdate.IsZero() {
		if now.Sub(resp.ThisUpdate) > maxAge {
			return fmt.Errorf("the next update is not specified and the update at %s is older than %s", resp.ThisUpdate.Format(time.RFC3339), maxAge)
		}
	} else if !now.Before(resp.NextUpdate) {
		return fmt.Errorf("expired at %s", resp.NextUpdate.Format(time.RFC3339))
	}

	return nil
}

type ocspCertID struct {
	HashAlgorithm pkix.AlgorithmIdentifier
	NameHash      []byte
	IssuerKeyHash []byte
	SerialNumber  *big.Int
}

type ocspResponseData struct {
	Version     int `asn1:"optional,default:0,explicit,tag:0"`
	ResponderID asn1.RawValue
	ProducedAt  time.Time `asn1:"generalized"`
	Responses   []struct {
		CertID ocspCertID
	}
}

func verifyOCSPCertID(resp *ocsp.Response, cert, issuer *x509.Certificate) (err error) {
	var (
		data ocspResponseData
		spki struct {
			Algorithm pkix.AlgorithmIdentifier
			PublicKey asn1.BitString
		}
	)

	if _, err = asn1.Unmarshal(resp.TBSResponseData, &data); err != nil {
		return fmt.Errorf("invalid response data: %w", err)
	}

	if _, err = asn1.Unmarshal(issuer.RawSubjectPublicKeyInfo, &spki); err != nil {
		return fmt.Errorf("invalid issuer public key: %w", err)
	}

	if !resp.IssuerHash.Available() {
		return errors.New("unsupported issuer hash algorithm")
	}

	for _, single := range data.Responses {
		if cert.SerialNumber.Cmp(single.CertID.SerialNumber) != 0 {
			continue
		}

		h := resp.IssuerHash.New()
		h.Write(issuer.RawSubject)

		if !bytes.Equal(h.Sum(nil), single.CertID.NameHash) {
			return errors.New("the issuer name hash does not match the certificate issuer")
		}

		h.Reset()
		h.Write(spki.PublicKey.RightAlign())

		if !bytes.Equal(h.Sum(nil), single.CertID.IssuerKeyHash) {
			return errors.New("the issuer key hash does not match the certificate issuer")
		}

		return nil
	}

	return errors.New("no response matching the supplied certificate")
}
