//go:build go1.27

package webauthncose

import (
	"crypto/x509"
	"encoding/asn1"
)

const (
	// AlgMLDSA44 is ML-DSA with parameter set ML-DSA-44 (FIPS 204). A credential using it carries an [AKP] key.
	//
	// This identifier is only declared when the library is built with Go 1.27 or newer, as the ML-DSA
	// implementation it verifies with is the standard library one. A Relying Party cannot name an algorithm a
	// build is unable to verify a signature with, so a request for it either compiles and works or does not
	// compile at all.
	//
	// Of the credential parameter lists the webauthn package provides, CredentialParametersPQCRecommendedL3 is
	// the only one which requests it, and requests all three parameter sets ahead of the classical algorithms. A
	// Relying Party using any of the other lists names it explicitly.
	AlgMLDSA44 COSEAlgorithmIdentifier = -48

	// AlgMLDSA65 is ML-DSA with parameter set ML-DSA-65 (FIPS 204). See [AlgMLDSA44] for the conditions under
	// which this library verifies with it.
	AlgMLDSA65 COSEAlgorithmIdentifier = -49

	// AlgMLDSA87 is ML-DSA with parameter set ML-DSA-87 (FIPS 204). See [AlgMLDSA44] for the conditions under
	// which this library verifies with it.
	AlgMLDSA87 COSEAlgorithmIdentifier = -50
)

// init registers the ML-DSA parameter sets in the tables the rest of the package reads, which are declared without
// them so that a build which cannot verify an ML-DSA signature does not describe one either.
//
// The hash in each detail is the zero value: ML-DSA signs the message rather than a digest of it, so there is no
// pre-hash to register. [HasherFromCOSEAlg] reports this as no hash rather than returning one which cannot be
// constructed. The x509 signature algorithms are what let the attestation formats verify an ML-DSA x5c chain, and the
// object identifiers are how an X.509 AlgorithmIdentifier states those same signature algorithms.
func init() {
	coseAlgorithmNames[AlgMLDSA44] = "ML-DSA-44"
	coseAlgorithmNames[AlgMLDSA65] = "ML-DSA-65"
	coseAlgorithmNames[AlgMLDSA87] = "ML-DSA-87"

	COSESignatureAlgorithmDetails[AlgMLDSA44] = COSESignatureAlgorithmDetail{name: "ML-DSA-44", sigAlg: x509.MLDSA44}
	COSESignatureAlgorithmDetails[AlgMLDSA65] = COSESignatureAlgorithmDetail{name: "ML-DSA-65", sigAlg: x509.MLDSA65}
	COSESignatureAlgorithmDetails[AlgMLDSA87] = COSESignatureAlgorithmDetail{name: "ML-DSA-87", sigAlg: x509.MLDSA87}

	coseAlgorithmObjectIdentifiers[AlgMLDSA44] = oidASN1SignatureAlgorithmMLDSA44
	coseAlgorithmObjectIdentifiers[AlgMLDSA65] = oidASN1SignatureAlgorithmMLDSA65
	coseAlgorithmObjectIdentifiers[AlgMLDSA87] = oidASN1SignatureAlgorithmMLDSA87
}

// The ASN.1 object identifiers of the ML-DSA parameter sets, registered by NIST in the Computer Security Objects
// Register under the nistAlgorithms arc. Unlike the classical algorithms, each parameter set has an identifier of
// its own and the AlgorithmIdentifier carries absent parameters, so the identifier settles which parameter set a
// signature uses on its own. The same identifier names the algorithm of a subjectPublicKeyInfo carrying a key of
// that parameter set.
//
// Registry: https://csrc.nist.gov/projects/computer-security-objects-register/algorithm-registration
var (
	oidASN1SignatureAlgorithmMLDSA44 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 17}
	oidASN1SignatureAlgorithmMLDSA65 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 18}
	oidASN1SignatureAlgorithmMLDSA87 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 19}
)
