package webauthncose

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"slices"
	"strconv"
)

// COSEAlgorithmIdentifier is a number identifying a cryptographic algorithm. The algorithm identifiers SHOULD be values
// registered in the IANA COSE Algorithms registry [https://www.w3.org/TR/webauthn/#biblio-iana-cose-algs-reg], for
// instance, -7 for "ES256" and -257 for "RS256".
//
// Specification: §5.8.5. Cryptographic Algorithm Identifier (https://www.w3.org/TR/webauthn/#sctn-alg-identifier)
type COSEAlgorithmIdentifier int

const (
	// AlgES256 ECDSA with SHA-256.
	AlgES256 COSEAlgorithmIdentifier = -7

	// AlgEdDSA EdDSA.
	AlgEdDSA COSEAlgorithmIdentifier = -8

	// AlgESP256 is ECDSA using P-256 curve with pre-hashed SHA-256 input.
	AlgESP256 COSEAlgorithmIdentifier = -9

	// AlgEd25519 is EdDSA using the Ed25519 curve specifically. Unlike [AlgEdDSA] which is the generic EdDSA
	// identifier, this explicitly specifies the Ed25519 curve.
	AlgEd25519 COSEAlgorithmIdentifier = -19

	// AlgES384 ECDSA with SHA-384.
	AlgES384 COSEAlgorithmIdentifier = -35

	// AlgES512 ECDSA with SHA-512.
	AlgES512 COSEAlgorithmIdentifier = -36

	// AlgPS256 RSASSA-PSS with SHA-256.
	AlgPS256 COSEAlgorithmIdentifier = -37

	// AlgPS384 RSASSA-PSS with SHA-384.
	AlgPS384 COSEAlgorithmIdentifier = -38

	// AlgPS512 RSASSA-PSS with SHA-512.
	AlgPS512 COSEAlgorithmIdentifier = -39

	// AlgES256K is ECDSA using secp256k1 curve and SHA-256. It is not requested by any of the credential parameter
	// lists this library provides, so a Relying Party that wants it must ask for it explicitly. An attestation
	// statement whose own algorithm is AlgES256K cannot convey an x5c chain, as certificates on this curve cannot
	// be parsed.
	AlgES256K COSEAlgorithmIdentifier = -47

	// The ML-DSA parameter sets occupy -48, -49 and -50. They are declared in the file which implements them, as
	// they are only usable when this library is built with Go 1.27 or newer.

	// AlgESP384 is ECDSA using P-384 curve with pre-hashed SHA-384 input.
	AlgESP384 COSEAlgorithmIdentifier = -51

	// AlgESP512 is ECDSA using P-521 curve with pre-hashed SHA-512 input.
	AlgESP512 COSEAlgorithmIdentifier = -52

	// AlgRS256 RSASSA-PKCS1-v1_5 with SHA-256.
	AlgRS256 COSEAlgorithmIdentifier = -257

	// AlgRS384 RSASSA-PKCS1-v1_5 with SHA-384.
	AlgRS384 COSEAlgorithmIdentifier = -258

	// AlgRS512 RSASSA-PKCS1-v1_5 with SHA-512.
	AlgRS512 COSEAlgorithmIdentifier = -259

	// AlgRS1 RSASSA-PKCS1-v1_5 with SHA-1.
	AlgRS1 COSEAlgorithmIdentifier = -65535
)

// String returns the short name under which the algorithm is registered, falling back to its numeric identifier for
// an algorithm this library does not model.
//
// This is deliberately distinct from the name member of [COSESignatureAlgorithmDetails], which describes the
// primitives the algorithm composes (i.e. "ECDSA-SHA256") rather than naming the algorithm itself.
//
// Registry: https://www.iana.org/assignments/cose/cose.xhtml#algorithms
func (a COSEAlgorithmIdentifier) String() string {
	if name, ok := coseAlgorithmNames[a]; ok {
		return name
	}

	return strconv.Itoa(int(a))
}

// coseAlgorithmNames maps each algorithm identifier this library models to the short name under which it is
// registered, backing [COSEAlgorithmIdentifier.String].
//
// Registry: https://www.iana.org/assignments/cose/cose.xhtml#algorithms
var coseAlgorithmNames = map[COSEAlgorithmIdentifier]string{
	AlgES256:   "ES256",
	AlgEdDSA:   "EdDSA",
	AlgESP256:  "ESP256",
	AlgEd25519: "Ed25519",
	AlgES384:   "ES384",
	AlgES512:   "ES512",
	AlgPS256:   "PS256",
	AlgPS384:   "PS384",
	AlgPS512:   "PS512",
	AlgES256K:  "ES256K",
	AlgESP384:  "ESP384",
	AlgESP512:  "ESP512",
	AlgRS256:   "RS256",
	AlgRS384:   "RS384",
	AlgRS512:   "RS512",
	AlgRS1:     "RS1",
}

// ObjectIdentifier returns the ASN.1 object identifier of the signature algorithm this COSE algorithm names, and
// whether this library has one registered for it. An algorithm it does not model, or one which names no signature
// algorithm an X.509 certificate can be signed with, yields no identifier rather than a substituted one.
//
// The mapping is many to one in three places, because an object identifier names a signature algorithm and nothing
// else. ESP256, ESP384 and ESP512 share theirs with ES256, ES384 and ES512, as what separates them is whether the
// Relying Party hands the algorithm a message or a digest of one rather than anything the signature itself records.
// ES256K shares the ECDSA with SHA-256 identifier, as the curve is stated by the key and not by the signature
// algorithm. PS256, PS384 and PS512 share id-RSASSA-PSS, as a PSS AlgorithmIdentifier states its digest in the
// parameters which follow the object identifier: a caller which must tell one from another has to read those
// parameters, and cannot settle it from what this returns.
//
// A caller comparing what this returns against an identifier read from a certificate should also note that RS1 has a
// second identifier, the ISO 1.3.14.3.2.29 alias, which names the same algorithm. This returns the PKCS #1 one, as
// that is what an encoder writes.
//
// The returned identifier is a copy. [asn1.ObjectIdentifier] is a slice, and the table backing this is shared by
// every caller.
//
// Registry: https://www.iana.org/assignments/cose/cose.xhtml#algorithms
func (a COSEAlgorithmIdentifier) ObjectIdentifier() (oid asn1.ObjectIdentifier, ok bool) {
	if oid, ok = coseAlgorithmObjectIdentifiers[a]; !ok {
		return nil, false
	}

	return slices.Clone(oid), true
}

// coseAlgorithmObjectIdentifiers maps each algorithm this library models to the ASN.1 object identifier of the
// signature algorithm it names, backing [COSEAlgorithmIdentifier.ObjectIdentifier]. The ML-DSA parameter sets are
// registered by the file which declares them.
//
// See [COSEAlgorithmIdentifier.ObjectIdentifier] for why several algorithms share an identifier.
var coseAlgorithmObjectIdentifiers = map[COSEAlgorithmIdentifier]asn1.ObjectIdentifier{
	AlgRS1:     oidASN1SignatureAlgorithmSHA1WithRSA,
	AlgRS256:   oidASN1SignatureAlgorithmSHA256WithRSA,
	AlgRS384:   oidASN1SignatureAlgorithmSHA384WithRSA,
	AlgRS512:   oidASN1SignatureAlgorithmSHA512WithRSA,
	AlgPS256:   oidASN1SignatureAlgorithmRSAPSS,
	AlgPS384:   oidASN1SignatureAlgorithmRSAPSS,
	AlgPS512:   oidASN1SignatureAlgorithmRSAPSS,
	AlgES256:   oidASN1SignatureAlgorithmSHA256WithECDSA,
	AlgESP256:  oidASN1SignatureAlgorithmSHA256WithECDSA,
	AlgES256K:  oidASN1SignatureAlgorithmSHA256WithECDSA,
	AlgES384:   oidASN1SignatureAlgorithmSHA384WithECDSA,
	AlgESP384:  oidASN1SignatureAlgorithmSHA384WithECDSA,
	AlgES512:   oidASN1SignatureAlgorithmSHA512WithECDSA,
	AlgESP512:  oidASN1SignatureAlgorithmSHA512WithECDSA,
	AlgEdDSA:   oidASN1SignatureAlgorithmEd25519,
	AlgEd25519: oidASN1SignatureAlgorithmEd25519,
}

// The ASN.1 object identifiers of the signature algorithms the modelled COSE algorithms name. The RSA identifiers are
// from RFC8017 Appendix A, the ECDSA identifiers from RFC5758 §3.2, and the Ed25519 identifier from RFC8410 §3.
var (
	oidASN1SignatureAlgorithmSHA1WithRSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}
	oidASN1SignatureAlgorithmRSAPSS          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	oidASN1SignatureAlgorithmSHA256WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidASN1SignatureAlgorithmSHA384WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidASN1SignatureAlgorithmSHA512WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
	oidASN1SignatureAlgorithmSHA256WithECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidASN1SignatureAlgorithmSHA384WithECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidASN1SignatureAlgorithmSHA512WithECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
	oidASN1SignatureAlgorithmEd25519         = asn1.ObjectIdentifier{1, 3, 101, 112}
)

// COSESignatureAlgorithmDetail describes the primitives a COSE signature algorithm composes.
//
// The type is named rather than declared inline on [COSESignatureAlgorithmDetails] so that an algorithm whose
// support depends on the toolchain the library is built with can be registered from the file which implements it.
type COSESignatureAlgorithmDetail struct {
	name   string
	hash   crypto.Hash
	sigAlg x509.SignatureAlgorithm
}

var COSESignatureAlgorithmDetails = map[COSEAlgorithmIdentifier]COSESignatureAlgorithmDetail{
	AlgRS1:     {"SHA1-RSA", crypto.SHA1, x509.SHA1WithRSA},
	AlgRS256:   {"SHA256-RSA", crypto.SHA256, x509.SHA256WithRSA},
	AlgRS384:   {"SHA384-RSA", crypto.SHA384, x509.SHA384WithRSA},
	AlgRS512:   {"SHA512-RSA", crypto.SHA512, x509.SHA512WithRSA},
	AlgPS256:   {"SHA256-RSAPSS", crypto.SHA256, x509.SHA256WithRSAPSS},
	AlgPS384:   {"SHA384-RSAPSS", crypto.SHA384, x509.SHA384WithRSAPSS},
	AlgPS512:   {"SHA512-RSAPSS", crypto.SHA512, x509.SHA512WithRSAPSS},
	AlgES256:   {"ECDSA-SHA256", crypto.SHA256, x509.ECDSAWithSHA256},
	AlgESP256:  {"ECDSA-SHA256-Prehashed", crypto.SHA256, x509.ECDSAWithSHA256},
	AlgES384:   {"ECDSA-SHA384", crypto.SHA384, x509.ECDSAWithSHA384},
	AlgESP384:  {"ECDSA-SHA384-Prehashed", crypto.SHA384, x509.ECDSAWithSHA384},
	AlgES512:   {"ECDSA-SHA512", crypto.SHA512, x509.ECDSAWithSHA512},
	AlgESP512:  {"ECDSA-SHA512-Prehashed", crypto.SHA512, x509.ECDSAWithSHA512},
	AlgES256K:  {"ECDSA-SHA256", crypto.SHA256, x509.ECDSAWithSHA256},
	AlgEdDSA:   {"EdDSA", crypto.SHA512, x509.PureEd25519},
	AlgEd25519: {"Ed25519", crypto.SHA512, x509.PureEd25519},
}

// coseAlgorithmCurves binds each elliptic curve algorithm this library verifies with to the curve §5.8.5 requires a
// key using that algorithm to specify.
//
// The specification requires the crv parameter for ES256, ES384, ES512 and EdDSA. It states no requirement for the
// fully specified algorithms ESP256, ESP384, ESP512 and Ed25519, because those identifiers name their curve
// themselves and leave nothing for the parameter to settle. ES256K is treated the same way for the same reason: the
// specification does not list it, and the identifier names secp256k1 on its own. A crv such a key does carry is
// still held to the curve its algorithm names, as a key which contradicts itself is malformed however the
// requirement is written.
//
// Specification: §5.8.5. Cryptographic Algorithm Identifier (https://www.w3.org/TR/webauthn-3/#sctn-alg-identifier)
var coseAlgorithmCurves = map[COSEAlgorithmIdentifier]coseAlgorithmCurve{
	AlgES256:   {curve: P256, required: true},
	AlgES384:   {curve: P384, required: true},
	AlgES512:   {curve: P521, required: true},
	AlgEdDSA:   {curve: Ed25519, required: true},
	AlgESP256:  {curve: P256},
	AlgESP384:  {curve: P384},
	AlgESP512:  {curve: P521},
	AlgEd25519: {curve: Ed25519},
	AlgES256K:  {curve: Secp256k1},
}
