package webauthn

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/go-webauthn/webauthn/protocol"
)

// RegistrationOption is a functional option that modifies the [protocol.PublicKeyCredentialCreationOptions] sent
// to the client during a registration ceremony. Use the With* functions in this package (i.e.
// [WithConveyancePreference], [WithExclusions], [WithAuthenticatorSelection]) to create registration options.
type RegistrationOption func(*protocol.PublicKeyCredentialCreationOptions)

// BeginRegistration generates a new set of registration data to be sent to the client and authenticator. To set a
// conditional mediation requirement for the registration see [WebAuthn.BeginMediatedRegistration].
func (webauthn *WebAuthn) BeginRegistration(user User, opts ...RegistrationOption) (creation *protocol.CredentialCreation, session *SessionData, err error) {
	return webauthn.BeginMediatedRegistration(user, protocol.MediationDefault, opts...)
}

// BeginMediatedRegistration is similar to [WebAuthn.BeginRegistration] however it also allows specifying a credential
// mediation requirement.
func (webauthn *WebAuthn) BeginMediatedRegistration(user User, mediation protocol.CredentialMediationRequirement, opts ...RegistrationOption) (creation *protocol.CredentialCreation, session *SessionData, err error) {
	if err = webauthn.Config.validate(); err != nil {
		return nil, nil, fmt.Errorf(errFmtConfigValidate, err)
	}

	var (
		challenge    protocol.URLEncodedBase64
		entityUserID any
	)

	if challenge, err = protocol.CreateChallenge(); err != nil {
		return nil, nil, err
	}

	if webauthn.Config.EncodeUserIDAsString {
		entityUserID = string(user.WebAuthnID())
	} else {
		entityUserID = protocol.URLEncodedBase64(user.WebAuthnID())
	}

	entityUser := protocol.UserEntity{
		ID:          entityUserID,
		DisplayName: user.WebAuthnDisplayName(),
		CredentialEntity: protocol.CredentialEntity{
			Name: user.WebAuthnName(),
		},
	}

	entityRelyingParty := protocol.RelyingPartyEntity{
		ID: webauthn.Config.RPID,
		CredentialEntity: protocol.CredentialEntity{
			Name: webauthn.Config.RPDisplayName,
		},
	}

	credentialParams := CredentialParametersDefault()

	creation = &protocol.CredentialCreation{
		Response: protocol.PublicKeyCredentialCreationOptions{
			RelyingParty:           entityRelyingParty,
			User:                   entityUser,
			Challenge:              challenge,
			Parameters:             credentialParams,
			AuthenticatorSelection: webauthn.Config.AuthenticatorSelection,
			Attestation:            webauthn.Config.AttestationPreference,
		},
		Mediation: mediation,
	}

	for _, opt := range opts {
		opt(&creation.Response)
	}

	if len(creation.Response.RelyingParty.ID) == 0 {
		return nil, nil, fmt.Errorf("error generating credential creation: the relying party id must be provided via the configuration or a functional option for a creation")
	} else if err = protocol.ValidateRPID(creation.Response.RelyingParty.ID); err != nil {
		return nil, nil, fmt.Errorf("error generating credential creation: the relying party id failed to validate as it's not a valid domain string with error: %w", err)
	}

	if len(creation.Response.RelyingParty.Name) == 0 {
		return nil, nil, fmt.Errorf("error generating credential creation: the relying party display name must be provided via the configuration or a functional option for a creation")
	}

	if len(creation.Response.Challenge) < protocol.MinimumChallengeLength {
		return nil, nil, fmt.Errorf("error generating credential creation: the challenge must be at least 16 bytes")
	}

	if creation.Response.Timeout == 0 {
		switch creation.Response.AuthenticatorSelection.UserVerification {
		case protocol.VerificationDiscouraged:
			creation.Response.Timeout = int(webauthn.Config.Timeouts.Registration.TimeoutUVD.Milliseconds())
		default:
			creation.Response.Timeout = int(webauthn.Config.Timeouts.Registration.Timeout.Milliseconds())
		}
	}

	session = &SessionData{
		Challenge:        creation.Response.Challenge.String(),
		RelyingPartyID:   creation.Response.RelyingParty.ID,
		UserID:           user.WebAuthnID(),
		UserVerification: creation.Response.AuthenticatorSelection.UserVerification,
		CredParams:       creation.Response.Parameters,
		Mediation:        creation.Mediation,
	}

	if webauthn.Config.Timeouts.Registration.Enforce {
		session.Expires = time.Now().Add(time.Millisecond * time.Duration(creation.Response.Timeout))
	}

	return creation, session, nil
}

// FinishRegistration takes the response from the authenticator and client and verify the credential against the user's
// credentials and session data.
//
// As with all Finish functions this function requires a [*http.Request] but you can perform the same steps with the
// [protocol.ParseCredentialCreationResponseBody] or [protocol.ParseCredentialCreationResponseBytes] which require an
// [io.Reader] or byte array respectively, you can also use an arbitrary [*protocol.ParsedCredentialCreationData] which is
// returned from all of these functions i.e. by implementing a custom parser. The [User], [*SessionData], and
// [*protocol.ParsedCredentialCreationData] can then be used with the [WebAuthn.CreateCredential] function.
func (webauthn *WebAuthn) FinishRegistration(user User, session SessionData, request *http.Request) (credential *Credential, err error) {
	parsedResponse, err := protocol.ParseCredentialCreationResponse(request)
	if err != nil {
		return nil, err
	}

	return webauthn.CreateCredential(user, session, parsedResponse)
}

// CreateCredential verifies a parsed response against the user's credentials and session data.
//
// If you wish to skip performing the step required to parse the [*protocol.ParsedCredentialCreationData] and
// you're using net/http then you can use [WebAuthn.FinishRegistration] instead.
func (webauthn *WebAuthn) CreateCredential(user User, session SessionData, parsedResponse *protocol.ParsedCredentialCreationData) (credential *Credential, err error) {
	if !bytes.Equal(user.WebAuthnID(), session.UserID) {
		return nil, protocol.ErrBadRequest.WithDetails("ID mismatch for User and Session")
	}

	if !session.Expires.IsZero() && session.Expires.Before(time.Now()) {
		return nil, protocol.ErrBadRequest.WithDetails("Session has Expired")
	}

	shouldVerifyUser := session.UserVerification == protocol.VerificationRequired
	shouldVerifyUserPresence := session.Mediation != protocol.MediationConditional

	var clientDataHash []byte

	if clientDataHash, err = parsedResponse.Verify(session.Challenge, webauthn.Config.RPID, webauthn.Config.RPOrigins, webauthn.Config.RPTopOrigins, webauthn.Config.RPTopOriginVerificationMode, webauthn.Config.RPAllowCrossOrigin, shouldVerifyUser, shouldVerifyUserPresence, webauthn.Config.MDS, session.CredParams); err != nil {
		return nil, err
	}

	if credential, err = NewCredential(clientDataHash, parsedResponse); err != nil {
		return nil, err
	}

	if err = ValidateFilteredCredential(credential, webauthn.Config.Filtering); err != nil {
		return nil, err
	}

	return credential, nil
}

// ValidateFilteredCredential applies the supplied [FilteringConfig] to a freshly-created [Credential]
// and returns a non-nil error when the credential violates any configured filtering rule (backup-eligibility
// prohibition, permitted-AAGUID allow-list, prohibited-AAGUID deny-list). A nil filtering argument is treated
// as "no filtering" and the function returns nil.
//
// The zero AAGUID ([uuid.Nil]) is never excluded by the permitted list, preserving the documented
// [FilteringConfig] contract for authenticators that report no AAGUID.
//
// This function is invoked automatically by [WebAuthn.CreateCredential] using the [Config.Filtering] value;
// relying parties may also call it directly (e.g. to pre-validate a credential before persistence) with any
// FilteringConfig value of their choosing.
//
// The credential argument must be non-nil.
func ValidateFilteredCredential(credential *Credential, filtering *FilteringConfig) (err error) {
	if filtering == nil {
		return nil
	}

	if credential == nil {
		return protocol.ErrBadRequest.WithInfo("Credential is nil")
	}

	if filtering.ProhibitBackupEligibility && credential.Flags.BackupEligible {
		return protocol.ErrPolicyRestriction.WithInfo("Credential is Backup Eligible")
	}

	var aaguid uuid.UUID

	if err = aaguid.UnmarshalBinary(credential.Authenticator.AAGUID); err != nil {
		return protocol.ErrBadRequest.WithInfo("The AAGUID of the credential is not a valid UUID")
	}

	if len(filtering.PermittedAAGUIDs) != 0 {
		var success = false

		if aaguid == uuid.Nil {
			success = true
		} else {
			for _, permitted := range filtering.PermittedAAGUIDs {
				if permitted == aaguid {
					success = true

					break
				}
			}
		}

		if !success {
			return protocol.ErrPolicyRestriction.WithInfo("Credential has an AAGUID which is not permitted")
		}
	}

	if len(filtering.ProhibitedAAGUIDs) != 0 {
		for _, prohibited := range filtering.ProhibitedAAGUIDs {
			if prohibited == aaguid {
				return protocol.ErrPolicyRestriction.WithInfo("Credential has an AAGUID which is prohibited")
			}
		}
	}

	return nil
}
