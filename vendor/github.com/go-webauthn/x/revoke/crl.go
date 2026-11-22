package revoke

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"
)

const clockSkew = 5 * time.Minute

var (
	oidExtensionAuthorityKeyID           = asn1.ObjectIdentifier{2, 5, 29, 35}
	oidExtensionIssuerAltName            = asn1.ObjectIdentifier{2, 5, 29, 18}
	oidExtensionCRLNumber                = asn1.ObjectIdentifier{2, 5, 29, 20}
	oidExtensionDeltaCRLIndicator        = asn1.ObjectIdentifier{2, 5, 29, 27}
	oidExtensionIssuingDistributionPoint = asn1.ObjectIdentifier{2, 5, 29, 28}
	oidExtensionReasonCode               = asn1.ObjectIdentifier{2, 5, 29, 21}
	oidExtensionInvalidityDate           = asn1.ObjectIdentifier{2, 5, 29, 24}
	oidExtensionCertificateIssuer        = asn1.ObjectIdentifier{2, 5, 29, 29}
)

type crlKey struct {
	uri    string
	issuer [sha256.Size]byte
}

type crlCache struct {
	mu      sync.Mutex
	entries map[crlKey]*x509.RevocationList
}

func (c *crlCache) get(key crlKey, now time.Time) *x509.RevocationList {
	c.mu.Lock()
	defer c.mu.Unlock()

	crl, ok := c.entries[key]
	if !ok {
		return nil
	}

	if !now.Before(crl.NextUpdate) {
		delete(c.entries, key)

		return nil
	}

	return crl
}

func (c *crlCache) put(key crlKey, crl *x509.RevocationList, now time.Time) {
	if !now.Before(crl.NextUpdate) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = map[crlKey]*x509.RevocationList{}
	}

	for k, v := range c.entries {
		if !now.Before(v.NextUpdate) {
			delete(c.entries, k)
		}
	}

	c.entries[key] = crl
}

type distributionPointName struct {
	FullName     []asn1.RawValue  `asn1:"optional,tag:0"`
	RelativeName pkix.RDNSequence `asn1:"optional,tag:1"`
}

type issuingDistributionPoint struct {
	DistributionPoint          distributionPointName `asn1:"optional,tag:0"`
	OnlyContainsUserCerts      bool                  `asn1:"optional,tag:1"`
	OnlyContainsCACerts        bool                  `asn1:"optional,tag:2"`
	OnlySomeReasons            asn1.BitString        `asn1:"optional,tag:3"`
	IndirectCRL                bool                  `asn1:"optional,tag:4"`
	OnlyContainsAttributeCerts bool                  `asn1:"optional,tag:5"`
}

func (c *checker) checkCRL(ctx context.Context, cert, issuer *x509.Certificate, uri string) (revoked bool, err error) {
	key := crlKey{uri: uri, issuer: sha256.Sum256(issuer.Raw)}
	now := time.Now()

	crl := c.crls.get(key, now)

	if crl == nil {
		if crl, err = c.fetchCRL(ctx, uri); err != nil {
			return false, err
		}

		now = time.Now()

		if err = verifyCRL(crl, issuer, now); err != nil {
			return false, fmt.Errorf("%w from %q: %w", ErrCRLInvalid, uri, err)
		}

		c.crls.put(key, crl, now)
	}

	var limited bool

	if limited, err = verifyCRLScope(crl, cert); err != nil {
		return false, fmt.Errorf("%w from %q: %w", ErrCRLInvalid, uri, err)
	}

	for _, entry := range crl.RevokedCertificateEntries {
		if cert.SerialNumber.Cmp(entry.SerialNumber) == 0 {
			return true, nil
		}
	}

	if limited {
		return false, fmt.Errorf("%w from %q: the scope is limited to some revocation reasons", ErrCRLInvalid, uri)
	}

	return false, nil
}

func (c *checker) fetchCRL(ctx context.Context, uri string) (crl *x509.RevocationList, err error) {
	var data []byte

	if data, err = c.fetch(ctx, http.MethodGet, uri, "", nil, crlRead, c.maxCRL); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedGetCRL, err)
	}

	if p, _ := pem.Decode(data); p != nil && p.Type == "X509 CRL" {
		data = p.Bytes
	}

	if crl, err = x509.ParseRevocationList(data); err != nil {
		return nil, fmt.Errorf("%w from %q: %w", ErrCRLInvalid, uri, err)
	}

	return crl, nil
}

func verifyCRL(crl *x509.RevocationList, issuer *x509.Certificate, now time.Time) (err error) {
	if !bytes.Equal(crl.RawIssuer, issuer.RawSubject) {
		return errors.New("the issuer name does not match the certificate issuer")
	}

	if err = crl.CheckSignatureFrom(issuer); err != nil {
		return err
	}

	if crl.ThisUpdate.After(now.Add(clockSkew)) {
		return fmt.Errorf("not valid until %s", crl.ThisUpdate.Format(time.RFC3339))
	}

	if crl.NextUpdate.IsZero() {
		return errors.New("the next update is not specified")
	}

	if !now.Before(crl.NextUpdate) {
		return fmt.Errorf("expired at %s", crl.NextUpdate.Format(time.RFC3339))
	}

	for _, ext := range crl.Extensions {
		switch {
		case ext.Id.Equal(oidExtensionDeltaCRLIndicator):
			return errors.New("delta CRLs are not supported")
		case ext.Id.Equal(oidExtensionAuthorityKeyID), ext.Id.Equal(oidExtensionIssuerAltName),
			ext.Id.Equal(oidExtensionCRLNumber), ext.Id.Equal(oidExtensionIssuingDistributionPoint):
			continue
		case ext.Critical:
			return fmt.Errorf("unsupported critical extension %s", ext.Id)
		}
	}

	for _, entry := range crl.RevokedCertificateEntries {
		for _, ext := range entry.Extensions {
			switch {
			case ext.Id.Equal(oidExtensionCertificateIssuer):
				return errors.New("indirect CRLs are not supported")
			case ext.Id.Equal(oidExtensionReasonCode), ext.Id.Equal(oidExtensionInvalidityDate):
				continue
			case ext.Critical:
				return fmt.Errorf("unsupported critical entry extension %s", ext.Id)
			}
		}
	}

	return nil
}

func verifyCRLScope(crl *x509.RevocationList, cert *x509.Certificate) (limited bool, err error) {
	for _, ext := range crl.Extensions {
		if !ext.Id.Equal(oidExtensionIssuingDistributionPoint) {
			continue
		}

		var (
			idp  issuingDistributionPoint
			rest []byte
		)

		if rest, err = asn1.Unmarshal(ext.Value, &idp); err != nil {
			return false, fmt.Errorf("invalid issuing distribution point: %w", err)
		} else if len(rest) != 0 {
			return false, errors.New("invalid issuing distribution point: trailing data")
		}

		switch {
		case idp.IndirectCRL:
			return false, errors.New("indirect CRLs are not supported")
		case idp.OnlyContainsAttributeCerts:
			return false, errors.New("the scope is limited to attribute certificates")
		case idp.OnlyContainsUserCerts && cert.IsCA:
			return false, errors.New("the scope is limited to end entity certificates")
		case idp.OnlyContainsCACerts && !cert.IsCA:
			return false, errors.New("the scope is limited to CA certificates")
		case len(idp.DistributionPoint.RelativeName) != 0:
			return false, errors.New("relative distribution point names are not supported")
		}

		if idp.OnlySomeReasons.BitLength != 0 {
			limited = true
		}

		if len(idp.DistributionPoint.FullName) == 0 {
			continue
		}

		if !slices.ContainsFunc(idp.DistributionPoint.FullName, func(name asn1.RawValue) bool {
			return name.Class == asn1.ClassContextSpecific && name.Tag == 6 && slices.Contains(cert.CRLDistributionPoints, string(name.Bytes))
		}) {
			return false, errors.New("the distribution point does not match any distribution point of the certificate")
		}
	}

	return limited, nil
}
