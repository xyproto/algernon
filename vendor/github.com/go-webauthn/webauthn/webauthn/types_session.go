package webauthn

import (
	"time"

	"github.com/go-webauthn/webauthn/protocol"
)

//go:generate msgp

//msgp:replace protocol.UserVerificationRequirement with:string
//msgp:replace protocol.AuthenticationExtensions with:map[string]any
//msgp:replace protocol.CredentialMediationRequirement with:string
//msgp:clearomitted

// SessionData is the data that must be stored by the Relying Party between the Begin and Finish steps of a WebAuthn
// ceremony. It contains the challenge and other parameters needed to verify the authenticator's response.
//
// The Relying Party must store this data securely and associate it with the user's session. It should not be
// modifiable by the client (i.e. store it server-side or in a signed, opaque cookie). After the ceremony completes,
// the session data should be discarded.
//
// Every field returned by the Begin* functions must be delivered to the matching Finish* / Validate* call with
// the same values; if anything is dropped or reshaped in transit, verification will fail. Treat [SessionData] as
// an atomic record between those two calls.
//
// For consolidated persistence guidance; recommended schema shape, required lookup columns, and the rules
// that also apply to [Credential] records; see the [Storage] section of the
// [github.com/go-webauthn/webauthn/webauthn] package documentation.
//
// [Storage]: https://pkg.go.dev/github.com/go-webauthn/webauthn/webauthn#hdr-Storage
type SessionData struct {
	Challenge            string    `json:"challenge" msg:"c"`
	RelyingPartyID       string    `json:"rpId,omitempty" msg:"r,omitempty"`
	UserID               []byte    `json:"user_id,omitempty" msg:"u,omitempty"`
	AllowedCredentialIDs [][]byte  `json:"allowed_credentials,omitempty" msg:"allow,omitempty"`
	Expires              time.Time `json:"expires" msg:"exp"`

	UserVerification protocol.UserVerificationRequirement    `json:"userVerification,omitempty" msg:"uv,omitempty"`
	Extensions       protocol.AuthenticationExtensions       `json:"extensions,omitempty" msg:"exts,omitempty"`
	CredParams       []protocol.CredentialParameter          `json:"credParams,omitempty" msg:"params,omitempty"`
	Mediation        protocol.CredentialMediationRequirement `json:"mediation,omitempty" msg:"cmr,omitempty"`
}
