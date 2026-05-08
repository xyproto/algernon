package webauthn

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

// CredentialParametersDefault returns the default list of acceptable credential algorithms. This includes ES256,
// ES384, ES512, RS256, RS384, RS512, PS256, PS384, PS512, and EdDSA. The order indicates preference.
func CredentialParametersDefault() []protocol.CredentialParameter {
	return []protocol.CredentialParameter{
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES512,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS512,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS512,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgEdDSA,
		},
	}
}

// CredentialParametersRecommendedL3 returns the WebAuthn Level 3 recommended credential algorithm list: EdDSA,
// ES256, and RS256 (in that order). This is the minimal set recommended by the specification for broad
// authenticator compatibility.
func CredentialParametersRecommendedL3() []protocol.CredentialParameter {
	return []protocol.CredentialParameter{
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgEdDSA,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS256,
		},
	}
}

// CredentialParametersExtendedL3 returns the WebAuthn Level 3 recommended credential algorithm list (EdDSA, ES256,
// RS256) extended with all other algorithms supported by this library (ES384, ES512, RS384, RS512, PS256, PS384,
// PS512). The Level 3 recommended algorithms appear first to indicate preference.
func CredentialParametersExtendedL3() []protocol.CredentialParameter {
	return []protocol.CredentialParameter{
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgEdDSA,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES512,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgRS512,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS256,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS384,
		},
		{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgPS512,
		},
	}
}
