package protocol

// NewSignalAllAcceptedCredentials creates a new SignalAllAcceptedCredentials struct that can simply be encoded with
// json.Marshal.
func NewSignalAllAcceptedCredentials(rpid string, user AllAcceptedCredentialsUser) *SignalAllAcceptedCredentials {
	if user == nil {
		return nil
	}

	credentials := user.WebAuthnCredentialIDs()

	ids := make([]URLEncodedBase64, len(credentials))

	for i, id := range credentials {
		ids[i] = id
	}

	return &SignalAllAcceptedCredentials{
		AllAcceptedCredentialIDs: ids,
		RPID:                     rpid,
		UserID:                   user.WebAuthnID(),
	}
}

// SignalAllAcceptedCredentials is a struct which represents the CDDL of the same name.
type SignalAllAcceptedCredentials struct {
	AllAcceptedCredentialIDs []URLEncodedBase64 `json:"allAcceptedCredentialIds"`
	RPID                     string             `json:"rpId"`
	UserID                   URLEncodedBase64   `json:"userId"`
}

// SignalCurrentUserDetails is a struct which represents the CDDL of the same name.
type SignalCurrentUserDetails struct {
	DisplayName string           `json:"displayName"`
	Name        string           `json:"name"`
	RPID        string           `json:"rpId"`
	UserID      URLEncodedBase64 `json:"userId"`
}

// SignalUnknownCredential is a struct which represents the CDDL of the same name.
type SignalUnknownCredential struct {
	CredentialID URLEncodedBase64 `json:"credentialId"`
	RPID         string           `json:"rpId"`
}

// AllAcceptedCredentialsUser is an interface that can be implemented by a user to provide information about their
// accepted credentials.
type AllAcceptedCredentialsUser interface {
	WebAuthnID() []byte
	WebAuthnCredentialIDs() [][]byte
}
