// Package revoke provides functionality for checking the validity of a cert. Specifically, the temporal validity of the
// certificate is checked first, then any CRL and OCSP url in the cert is checked. This is a fork of the
// github.com/cloudflare/cfssl/revoke package. It's used to lookup the revocation status of X.509 Certificates.
//
// Revocation data is only trusted when it is authenticated by the issuer of the certificate being checked. Callers
// should verify the certificate chain first and then use VerifyCertificateWithIssuer or a Verifier with each
// certificate and its issuer from the verified chain. All requests are bound by a timeout and a response size limit.
package revoke
