package webauthn

import "github.com/go-webauthn/webauthn/protocol"

// WithCredentialParameters adjusts the credential parameters in the registration options.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialcreationoptions-pubkeycredparams)
func WithCredentialParameters(credentialParams []protocol.CredentialParameter) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.Parameters = credentialParams
	}
}

// WithExclusions adjusts the non-default parameters regarding credentials to exclude from registration.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialcreationoptions-excludecredentials)
func WithExclusions(excludeList []protocol.CredentialDescriptor) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.CredentialExcludeList = excludeList
	}
}

// WithAuthenticatorSelection adjusts the non-default parameters regarding the authenticator to select during
// registration.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialcreationoptions-authenticatorselection)
//
// Specification: §5.4.4. Authenticator Selection Criteria (https://www.w3.org/TR/webauthn/#dictdef-authenticatorselectioncriteria)
func WithAuthenticatorSelection(authenticatorSelection protocol.AuthenticatorSelection) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.AuthenticatorSelection = authenticatorSelection
	}
}

// WithResidentKeyRequirement sets both the resident key and require resident key protocol options.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialcreationoptions-authenticatorselection)
//
// Specification: §5.4.4. Authenticator Selection Criteria (https://www.w3.org/TR/webauthn/#dictdef-authenticatorselectioncriteria)
func WithResidentKeyRequirement(requirement protocol.ResidentKeyRequirement) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.AuthenticatorSelection.ResidentKey = requirement

		switch requirement {
		case protocol.ResidentKeyRequirementRequired:
			cco.AuthenticatorSelection.RequireResidentKey = protocol.ResidentKeyRequired()
		default:
			cco.AuthenticatorSelection.RequireResidentKey = protocol.ResidentKeyNotRequired()
		}
	}
}

// WithPublicKeyCredentialHints adjusts the non-default hints for credential types to select during registration.
//
// WebAuthn Level 3.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn-3/#dom-publickeycredentialcreationoptions-hints)
func WithPublicKeyCredentialHints(hints []protocol.PublicKeyCredentialHints) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.Hints = hints
	}
}

// WithConveyancePreference adjusts the non-default parameters regarding whether the authenticator should attest to the
// credential.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialcreationoptions-attestation)
func WithConveyancePreference(preference protocol.ConveyancePreference) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.Attestation = preference
	}
}

// WithAttestationFormats adjusts the non-default formats for credential types to select during registration.
//
// WebAuthn Level 3.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn-3/#dom-publickeycredentialcreationoptions-attestationformats)
func WithAttestationFormats(formats []protocol.AttestationFormat) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.AttestationFormats = formats
	}
}

// WithExtensions adjusts the extension parameter in the registration options.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn-3/#dom-publickeycredentialcreationoptions-extensions)
//
// Specification: §9. Extensions (https://www.w3.org/TR/webauthn/#webauthn-extensions)
func WithExtensions(extension protocol.AuthenticationExtensions) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.Extensions = extension
	}
}

// WithAppIdExcludeExtension automatically includes the specified appid if the CredentialExcludeList contains a credential
// with the type `fido-u2f`.
//
// Specification: §5.4. Parameters for Credential Generation (https://www.w3.org/TR/webauthn-3/#dom-publickeycredentialcreationoptions-extensions)
//
// Specification: §9. Extensions (https://www.w3.org/TR/webauthn/#webauthn-extensions)
//
// Specification: §10.1.2. FIDO AppID Exclusion Extension (https://www.w3.org/TR/webauthn/#sctn-appid-exclude-extension)
func WithAppIdExcludeExtension(appid string) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		for _, credential := range cco.CredentialExcludeList {
			if credential.AttestationFormat == string(protocol.AttestationFormatFIDOUniversalSecondFactor) {
				if cco.Extensions == nil {
					cco.Extensions = map[string]any{}
				}

				cco.Extensions[protocol.ExtensionAppIDExclude] = appid

				break
			}
		}
	}
}

// WithRegistrationRelyingPartyID sets the relying party id for the registration.
func WithRegistrationRelyingPartyID(id string) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.RelyingParty.ID = id
	}
}

// WithRegistrationRelyingPartyName sets the relying party name for the registration.
func WithRegistrationRelyingPartyName(name string) RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.RelyingParty.Name = name
	}
}
