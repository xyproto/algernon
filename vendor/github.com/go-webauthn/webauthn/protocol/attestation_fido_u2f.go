package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"fmt"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

// attestationFormatValidationHandlerFIDOU2F is the handler for the FIDO U2F Attestation Statement Format.
//
// The syntax of a FIDO U2F attestation statement is defined as follows:
//
// $$attStmtType //= (
//
//	    fmt: "fido-u2f",
//	    attStmt: u2fStmtFormat
//	)
//
//	u2fStmtFormat = {
//	                    x5c: [ attestnCert: bytes ],
//	                    sig: bytes
//	                }
//
// Specification: §8.6. FIDO U2F Attestation Statement Format
//
// See: https://www.w3.org/TR/webauthn/#sctn-fido-u2f-attestation
func attestationFormatValidationHandlerFIDOU2F(att AttestationObject, clientDataHash []byte, _ metadata.Provider) (attestationType string, x5cs []any, err error) {
	// Signing procedure. Non-normative verification procedure of expected requirement.
	// If the credential public key of the attested credential is not of algorithm -7 ("ES256"), stop and return an error.
	var key webauthncose.EC2PublicKeyData
	if err = webauthncbor.Unmarshal(att.AuthData.AttData.CredentialPublicKey, &key); err != nil {
		return "", nil, ErrAttestationCertificate.WithDetails("Error parsing public key").WithError(err)
	}

	if webauthncose.COSEAlgorithmIdentifier(key.Algorithm) != webauthncose.AlgES256 {
		return "", nil, ErrUnsupportedAlgorithm.WithDetails("Non-ES256 Public Key algorithm used")
	}

	var (
		sig []byte
		raw []byte
		x5c []any
		ok  bool
	)

	// Step 1. Verify that attStmt is valid CBOR conforming to the syntax defined above and perform CBOR decoding on it
	// to extract the contained fields.

	// Check for "x5c" which is a single element array containing the attestation certificate in X.509 format.
	if x5c, ok = att.AttStatement[stmtX5C].([]any); !ok {
		return "", nil, ErrAttestationFormat.WithDetails("Missing properly formatted x5c data")
	}

	// Note: Packed Attestation, FIDO U2F Attestation, and Assertion Signatures require ASN.1 DER sig values, but it is
	// RECOMMENDED that any new attestation formats defined not use ASN.1 encodings, but instead represent signatures as
	// equivalent fixed-length byte arrays without internal structure, using the same representations as used by COSE
	// signatures as defined in [RFC9053](https://www.rfc-editor.org/rfc/rfc9053.html) and
	// [RFC8230](https://www.rfc-editor.org/rfc/rfc8230.html).
	// This is described in §6.5.5 https://www.w3.org/TR/webauthn-3/#sctn-signature-attestation-types.

	// Check for "sig" which is The attestation signature. The signature was calculated over the (raw) U2F
	// registration response message https://www.w3.org/TR/webauthn/#biblio-fido-u2f-message-formats]
	// received by the client from the authenticator.
	if sig, ok = att.AttStatement[stmtSignature].([]byte); !ok {
		return "", nil, ErrAttestationFormat.WithDetails("Missing sig data")
	}

	// Step 2.
	//	 1. Check that x5c has exactly one element and let attCert be that element.
	//	 2. Let certificate public key be the public key conveyed by attCert.
	//	 3. If certificate public key is not an Elliptic Curve (EC) public key over the P-256 curve, terminate this
	//	    algorithm and return an appropriate error.

	// Step 2.1.
	if len(x5c) != 1 {
		return "", nil, ErrAttestationFormat.WithDetails("x5c must contain exactly one element")
	}

	// Step 2.2.
	if raw, ok = x5c[0].([]byte); !ok {
		return "", nil, ErrAttestationFormat.WithDetails("Error decoding ASN.1 data from x5c")
	}

	attCert, err := x509.ParseCertificate(raw)
	if err != nil {
		return "", nil, ErrAttestationFormat.WithDetails("Error parsing certificate from ASN.1 data into certificate").WithError(err)
	}

	// Step 2.3.
	if attCert.PublicKeyAlgorithm != x509.ECDSA {
		return "", nil, ErrAttestationFormat.WithDetails("Attestation certificate public key algorithm is not ECDSA")
	}

	// Step 3. Extract the claimed rpIdHash from authenticatorData, and the claimed credentialId and credentialPublicKey
	// from authenticatorData.attestedCredentialData.
	rpIdHash := att.AuthData.RPIDHash
	credentialID := att.AuthData.AttData.CredentialID

	// Step 4. Convert the COSE_KEY formatted credentialPublicKey (see Section 7 of RFC8152 [https://www.w3.org/TR/webauthn/#biblio-rfc8152])
	// to Raw ANSI X9.62 public key format (see ALG_KEY_ECC_X962_RAW in Section 3.6.2 Public Key
	// Representation Formats of
	// [FIDO-Registry](https://fidoalliance.org/specs/fido-v2.0-id-20180227/fido-registry-v2.0-id-20180227.html#public-key-representation-formats)).

	// Let x be the value corresponding to the "-2" key (representing x coordinate) in credentialPublicKey, and confirm
	// its size to be of 32 bytes. If size differs or "-2" key is not found, terminate this algorithm and return an
	// appropriate error.

	// Let y be the value corresponding to the "-3" key (representing y coordinate) in credentialPublicKey, and confirm
	// its size to be of 32 bytes. If size differs or "-3" key is not found, terminate this algorithm and return an
	// appropriate error.
	credentialPublicKey, ok := attCert.PublicKey.(*ecdsa.PublicKey)
	if !ok || credentialPublicKey.Curve != elliptic.P256() {
		return "", nil, ErrAttestationFormat.WithDetails("Attestation certificate does not contain a P-256 ECDSA public key")
	}

	if len(key.XCoord) != 32 || len(key.YCoord) != 32 {
		return "", nil, ErrAttestation.WithDetails("X or Y Coordinate for key is invalid length")
	}

	// Let publicKeyU2F be the concatenation 0x04 || x || y.
	publicKeyU2F := bytes.NewBuffer([]byte{0x04})
	publicKeyU2F.Write(key.XCoord)
	publicKeyU2F.Write(key.YCoord)

	// Step 5. Let verificationData be the concatenation of (0x00 || rpIdHash || clientDataHash || credentialId || publicKeyU2F)
	// (see Section 4.3 of [FIDO-U2F-Message-Formats](https://fidoalliance.org/specs/fido-u2f-v1.1-id-20160915/fido-u2f-raw-message-formats-v1.1-id-20160915.html#registration-response-message-success)).
	verificationData := bytes.NewBuffer([]byte{0x00})
	verificationData.Write(rpIdHash)
	verificationData.Write(clientDataHash)
	verificationData.Write(credentialID)
	verificationData.Write(publicKeyU2F.Bytes())

	// Step 6. Verify the sig using verificationData and the certificate public key per section 4.1.4 of [SEC1] with
	// SHA-256 as the hash function used in step two.
	if err = attCert.CheckSignature(x509.ECDSAWithSHA256, verificationData.Bytes(), sig); err != nil {
		return "", nil, ErrInvalidAttestation.WithDetails(fmt.Sprintf("Signature validation error: %+v", err)).WithError(err)
	}

	// TODO: Step 7. Optionally, inspect x5c and consult externally provided knowledge to determine whether attStmt
	//       conveys a Basic or AttCA attestation.

	// Step 8. If successful, return implementation-specific values representing attestation type Basic, AttCA or
	// uncertainty, and attestation trust path x5c.
	return string(metadata.BasicFull), x5c, nil
}

func init() {
	RegisterAttestationFormat(AttestationFormatFIDOUniversalSecondFactor, attestationFormatValidationHandlerFIDOU2F)
}
