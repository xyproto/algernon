package webauthn

import (
	"github.com/go-webauthn/webauthn/protocol"
)

//go:generate msgp

//msgp:replace protocol.AuthenticatorAttachment with:string
//msgp:clearomitted

// Authenticator represents a specific authenticator in the context of a [Credential].
type Authenticator struct {
	// The AAGUID of the authenticator. An AAGUID is defined as an array containing the globally unique
	// identifier of the authenticator model being sought.
	AAGUID []byte `json:"AAGUID,omitempty" msg:"aaguid,omitempty"`

	// SignCount is a representation of the number of times the Authenticator or Credential have been used to login.
	// Upon a new login operation, the Relying Party compares the stored signature counter value with the new SignCount
	// value returned in the assertion’s authenticator data. If this new SignCount value is less than or equal to the
	// stored value, a cloned authenticator may exist, or the authenticator may be malfunctioning.
	SignCount uint32 `json:"signCount,omitempty" msg:"sc,omitempty"`

	// CloneWarning is a signal that the authenticator may be cloned, i.e. at least two copies of the
	// credential private key may exist and are being used in parallel. Relying Parties should incorporate
	// this information into their risk scoring. Whether the Relying Party updates the stored signature
	// counter value in this case, or not, or fails the authentication ceremony or not, is Relying Party-specific.
	CloneWarning bool `json:"cloneWarning,omitempty" msg:"cw,omitempty"`

	// Attachment is the authenticatorAttachment value returned by the request.
	Attachment protocol.AuthenticatorAttachment `json:"attachment,omitempty" msg:"aa,omitempty"`
}

// SelectAuthenticator is a convenience function that constructs a [protocol.AuthenticatorSelection] from individual
// string and boolean parameters. Use [protocol.ResidentKeyRequired] or [protocol.ResidentKeyNotRequired] for the rrk
// parameter.
func SelectAuthenticator(att string, rrk *bool, uv string) protocol.AuthenticatorSelection {
	return protocol.AuthenticatorSelection{
		AuthenticatorAttachment: protocol.AuthenticatorAttachment(att),
		RequireResidentKey:      rrk,
		UserVerification:        protocol.UserVerificationRequirement(uv),
	}
}

// UpdateCounter updates the authenticator and either sets the clone warning value or the sign count.
//
// Step 17 of §7.2. about verifying attestation. If the signature counter value authData.signCount
// is nonzero or the value stored in conjunction with credential’s id attribute is nonzero, then
// run the following sub-step:
//
//	If the signature counter value authData.signCount is
//
//	→ Greater than the signature counter value stored in conjunction with credential’s id attribute.
//	Update the stored signature counter value, associated with credential’s id attribute, to be the value of
//	authData.signCount.
//
//	→ Less than or equal to the signature counter value stored in conjunction with credential’s id attribute.
//	This is a signal that the authenticator may be cloned, see CloneWarning above for more information.
func (a *Authenticator) UpdateCounter(authDataCount uint32) {
	if authDataCount <= a.SignCount && (authDataCount != 0 || a.SignCount != 0) {
		a.CloneWarning = true

		return
	}

	a.SignCount = authDataCount
}
