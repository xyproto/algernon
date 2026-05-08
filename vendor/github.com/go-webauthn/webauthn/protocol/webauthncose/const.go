package webauthncose

const (
	keyCannotDisplay = "Cannot display key"
)

const ecCoordSize = 32

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

	// AlgES256K is ECDSA using secp256k1 curve and SHA-256.
	AlgES256K COSEAlgorithmIdentifier = -47

	// AlgMLDSA44 is ML-DSA with parameter set ML-DSA-44 (FIPS 204).
	AlgMLDSA44 COSEAlgorithmIdentifier = -48

	// AlgMLDSA65 is ML-DSA with parameter set ML-DSA-65 (FIPS 204).
	AlgMLDSA65 COSEAlgorithmIdentifier = -49

	// AlgMLDSA87 is ML-DSA with parameter set ML-DSA-87 (FIPS 204).
	AlgMLDSA87 COSEAlgorithmIdentifier = -50

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

// COSEKeyType is The Key type derived from the IANA COSE AuthData.
type COSEKeyType int

const (
	// KeyTypeReserved is a reserved value.
	KeyTypeReserved COSEKeyType = iota

	// OctetKey is an Octet Key.
	OctetKey

	// EllipticKey is an Elliptic Curve Public Key.
	EllipticKey

	// RSAKey is an RSA Public Key.
	RSAKey

	// Symmetric Keys.
	Symmetric

	// HSSLMS is the public key for HSS/LMS hash-based digital signature.
	HSSLMS

	// WalnutDSA is the public key for Walnut Digital Signature Algorithm.
	WalnutDSA

	// AKP is the key type for algorithm key pairs (i.e. ML-DSA).
	AKP
)

// COSEEllipticCurve is an enumerator that represents the COSE Elliptic Curves.
//
// Specification: https://www.iana.org/assignments/cose/cose.xhtml#elliptic-curves
type COSEEllipticCurve int

const (
	// EllipticCurveReserved is the COSE EC Reserved value.
	EllipticCurveReserved COSEEllipticCurve = iota

	// P256 represents NIST P-256 also known as secp256r1.
	P256

	// P384 represents NIST P-384 also known as secp384r1.
	P384

	// P521 represents NIST P-521 also known as secp521r1.
	P521

	// X25519 for use w/ ECDH only.
	X25519

	// X448 for use w/ ECDH only.
	X448

	// Ed25519 for use w/ EdDSA only.
	Ed25519

	// Ed448 for use w/ EdDSA only.
	Ed448

	// Secp256k1 is the SECG secp256k1 curve.
	Secp256k1
)
