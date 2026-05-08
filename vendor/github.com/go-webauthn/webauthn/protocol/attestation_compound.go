package protocol

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/go-webauthn/webauthn/metadata"
)

func init() {
	RegisterAttestationFormat(AttestationFormatCompound, attestationFormatValidationHandlerCompound)
}

// attestationFormatValidationHandlerCompound is the handler for the Compound Attestation Statement Format.
//
// The syntax of a Compound Attestation statement is defined by the following CDDL:
//
// $$attStmtType //= (
//
//	    fmt: "compound",
//	    attStmt: [2* nonCompoundAttStmt]
//	)
//
// nonCompoundAttStmt = { $$attStmtType } .within { fmt: text .ne "compound", * any => any }
//
// Specification: §8.9. Compound Attestation Statement Forma
//
// See: https://www.w3.org/TR/webauthn-3/#sctn-compound-attestation
//
//nolint:gocyclo
func attestationFormatValidationHandlerCompound(att AttestationObject, clientDataHash []byte, mds metadata.Provider) (attestationType string, x5cs []any, err error) {
	var (
		aaguid   uuid.UUID
		raw      any
		ok       bool
		stmts    []any
		subStmt  map[string]any
		attStmts []NonCompoundAttestationObject
	)

	if len(att.AuthData.AttData.AAGUID) != 0 {
		if aaguid, err = uuid.FromBytes(att.AuthData.AttData.AAGUID); err != nil {
			return "", nil, ErrInvalidAttestation.WithInfo("Error occurred parsing AAGUID during attestation validation").WithDetails(err.Error()).WithError(err)
		}
	}

	if raw, ok = att.AttStatement[stmtAttStmt]; !ok {
		return "", nil, ErrInvalidAttestation.WithDetails("Compound statement missing attStmt")
	}

	if stmts, ok = raw.([]any); !ok {
		return "", nil, ErrInvalidAttestation.WithDetails("Compound statement attStmt isn't an array")
	}

	if len(stmts) < 2 {
		return "", nil, ErrInvalidAttestation.WithDetails("Compound statement attStmt isn't an array with at least two other statements")
	}

	for _, stmt := range stmts {
		if subStmt, ok = stmt.(map[string]any); !ok {
			return "", nil, ErrInvalidAttestation.WithDetails("Compound statement attStmt contains one or more items that isn't an object")
		}

		var attStmt NonCompoundAttestationObject

		if attStmt.Format, ok = subStmt[stmtFmt].(string); !ok {
			return "", nil, ErrInvalidAttestation.WithDetails("Compound sub-statement does not have a format")
		}

		if attStmt.AttStatement, ok = subStmt[stmtAttStmt].(map[string]any); !ok {
			return "", nil, ErrInvalidAttestation.WithDetails("Compound sub-statement does not have an attestation statement")
		}

		switch AttestationFormat(attStmt.Format) {
		case AttestationFormatCompound:
			return "", nil, ErrInvalidAttestation.WithDetails("Compound sub-statement has a format of compound which is not allowed")
		case "":
			return "", nil, ErrInvalidAttestation.WithDetails("Compound sub-statement has an empty format which is not allowed")
		default:
			if _, ok = attestationRegistry[AttestationFormat(attStmt.Format)]; !ok {
				return "", nil, ErrAttestationFormat.WithInfo(fmt.Sprintf("Attestation sub-statement format %s is unsupported", attStmt.Format))
			}

			attStmts = append(attStmts, attStmt)
		}
	}

	for _, attStmt := range attStmts {
		object := AttestationObject{
			Format:       attStmt.Format,
			AttStatement: attStmt.AttStatement,
			AuthData:     att.AuthData,
			RawAuthData:  att.RawAuthData,
		}

		var (
			cx5cs      []any
			subAttType string
		)

		if subAttType, cx5cs, err = attestationRegistry[AttestationFormat(object.Format)](object, clientDataHash, mds); err != nil {
			return "", nil, err
		}

		if mds == nil {
			continue
		}

		if e := ValidateMetadata(context.Background(), mds, aaguid, subAttType, object.Format, cx5cs); e != nil {
			return "", nil, ErrInvalidAttestation.WithInfo(fmt.Sprintf("Error occurred validating metadata during attestation validation: %+v", e)).WithDetails(e.DevInfo).WithError(e)
		}
	}

	return stmtTypNone, nil, nil
}
