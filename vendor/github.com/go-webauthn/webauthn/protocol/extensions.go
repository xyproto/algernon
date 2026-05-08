package protocol

// Extensions are discussed in §9. WebAuthn Extensions (https://www.w3.org/TR/webauthn/#extensions).

// For a list of commonly supported extensions, see §10. Defined Extensions
// (https://www.w3.org/TR/webauthn/#sctn-defined-extensions).

// AuthenticationExtensionsClientOutputs represents the IDL of the same name. It is a map of extension identifier
// strings to their output values, returned by the client after a create() or get() call.
//
// Specification: §5.9. Authentication Extensions Client Outputs (https://www.w3.org/TR/webauthn/#iface-authentication-extensions-client-outputs)
type AuthenticationExtensionsClientOutputs map[string]any

const (
	// ExtensionAppID is the FIDO AppID Extension identifier. It is used during authentication to allow credentials
	// registered via the legacy FIDO U2F JavaScript API to be used with WebAuthn.
	//
	// Specification: §10.1. FIDO AppID Extension (https://www.w3.org/TR/webauthn/#sctn-appid-extension)
	ExtensionAppID = "appid"

	// ExtensionAppIDExclude is the FIDO AppID Exclusion Extension identifier. It is used during registration to
	// exclude credentials previously registered via the legacy FIDO U2F JavaScript API.
	//
	// Specification: §10.2. FIDO AppID Exclusion Extension (https://www.w3.org/TR/webauthn/#sctn-appid-exclude-extension)
	ExtensionAppIDExclude = "appidExclude"
)
