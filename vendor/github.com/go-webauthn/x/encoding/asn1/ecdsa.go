package asn1

import (
	"fmt"
	"math/big"
)

// ECDSASignature is the ASN.1 SEQUENCE which carries the two integers of an ECDSA signature.
//
// Specification: RFC 3279 §2.2.3. ECDSA Signature Algorithm (https://datatracker.ietf.org/doc/html/rfc3279#section-2.2.3)
type ECDSASignature struct {
	R, S *big.Int
}

// NormalizeECDSASignature decodes an ECDSA signature which encodes its integers under BER and returns the DER
// encoding of the same signature, so that a verifier which accepts only DER can be given a signature an
// authenticator produced under the more permissive rules.
//
// The relaxation is confined to the minimal encoding requirement DER places on an INTEGER, which is the deviation
// [WithUnmarshalAllowBERIntegers] permits and the only one observed from authenticators in practice. Every other
// requirement of DER is still enforced: a non-minimal length, an indefinite length, and a value which is not a
// SEQUENCE of exactly two integers are all rejected, as is any data trailing either the signature or the two
// integers within it. An integer which is not positive is rejected as it cannot be a component of an ECDSA
// signature.
//
// The result is a function of the two decoded integers alone. Re-encoding from [big.Int] emits the minimal form of
// each one, so two signatures which differ only in the padding of their integers normalize to identical bytes, and
// a signature which is already DER normalizes to itself. There is no fallback: an input this function cannot decode
// under the rules above yields an error rather than a signature which is passed on to be verified.
//
// Accepting a non-DER signature makes the encoding of a signature malleable, in that byte sequences which are not
// equal verify against the same message. Callers to which the encoding of a signature is material, rather than only
// the pair of integers it carries, should not use this function.
func NormalizeECDSASignature(sig []byte) (der []byte, err error) {
	var (
		sequence  RawValue
		signature ECDSASignature
		rest      []byte
	)

	if rest, err = Unmarshal(sig, &sequence); err != nil {
		return nil, err
	}

	if len(rest) != 0 {
		return nil, StructuralError{fmt.Sprintf("ecdsa signature has %d bytes of trailing data", len(rest))}
	}

	if sequence.Class != ClassUniversal || sequence.Tag != TagSequence || !sequence.IsCompound {
		return nil, StructuralError{"ecdsa signature is not a sequence"}
	}

	// The two integers are decoded individually because unmarshalling the sequence into a struct discards any
	// element which trails the fields it models, which would silently accept a sequence of more than two of them.
	rest = sequence.Bytes

	if rest, err = Unmarshal(rest, &signature.R, WithUnmarshalAllowBERIntegers(true)); err != nil {
		return nil, err
	}

	if rest, err = Unmarshal(rest, &signature.S, WithUnmarshalAllowBERIntegers(true)); err != nil {
		return nil, err
	}

	if len(rest) != 0 {
		return nil, StructuralError{fmt.Sprintf("ecdsa signature sequence has %d bytes of trailing data", len(rest))}
	}

	if signature.R.Sign() <= 0 {
		return nil, StructuralError{"ecdsa signature has a r value which is not a positive integer"}
	}

	if signature.S.Sign() <= 0 {
		return nil, StructuralError{"ecdsa signature has a s value which is not a positive integer"}
	}

	return Marshal(signature)
}
