package protocol

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

// The CredentialAssertionResponse is the raw response returned to the Relying Party from an authenticator when we request a
// credential for login/assertion.
type CredentialAssertionResponse struct {
	PublicKeyCredential

	AssertionResponse AuthenticatorAssertionResponse `json:"response"`
}

// The ParsedCredentialAssertionData is the parsed [CredentialAssertionResponse] that has been marshalled into a format
// that allows us to verify the client and authenticator data inside the response.
type ParsedCredentialAssertionData struct {
	ParsedPublicKeyCredential

	Response ParsedAssertionResponse
	Raw      CredentialAssertionResponse
}

// The AuthenticatorAssertionResponse contains the raw authenticator assertion data and is parsed into
// [ParsedAssertionResponse].
type AuthenticatorAssertionResponse struct {
	AuthenticatorResponse

	AuthenticatorData URLEncodedBase64 `json:"authenticatorData"`
	Signature         URLEncodedBase64 `json:"signature"`
	UserHandle        URLEncodedBase64 `json:"userHandle,omitempty"`
}

// ParsedAssertionResponse is the parsed form of [AuthenticatorAssertionResponse].
type ParsedAssertionResponse struct {
	CollectedClientData CollectedClientData
	AuthenticatorData   AuthenticatorData
	Signature           []byte
	UserHandle          []byte
}

// ParseCredentialRequestResponse parses a login/assertion response from a [*http.Request]. The request body is
// automatically drained and closed after parsing.
//
// This is the standard entry point when using [net/http]. For implementations that don't use [net/http], see
// [ParseCredentialRequestResponseBody] (accepts an [io.Reader]) or [ParseCredentialRequestResponseBytes] (accepts a
// []byte).
func ParseCredentialRequestResponse(response *http.Request) (*ParsedCredentialAssertionData, error) {
	if response == nil || response.Body == nil {
		return nil, ErrBadRequest.WithDetails("No response given")
	}

	defer func(request *http.Request) {
		_, _ = io.Copy(io.Discard, request.Body)
		_ = request.Body.Close()
	}(response)

	return ParseCredentialRequestResponseBody(response.Body)
}

// ParseCredentialRequestResponseBody parses a login/assertion response from an [io.Reader]. The caller is responsible
// for closing the reader if applicable.
//
// This is the framework-agnostic variant of [ParseCredentialRequestResponse]. For a [*http.Request] use
// [ParseCredentialRequestResponse] instead. For raw bytes use [ParseCredentialRequestResponseBytes].
func ParseCredentialRequestResponseBody(body io.Reader) (par *ParsedCredentialAssertionData, err error) {
	var car CredentialAssertionResponse

	if err = decodeBody(body, &car); err != nil {
		return nil, ErrBadRequest.WithDetails("Parse error for Assertion").WithInfo(err.Error()).WithError(err)
	}

	return car.Parse()
}

// ParseCredentialRequestResponseBytes parses a login/assertion response from raw bytes.
//
// See also [ParseCredentialRequestResponse] (for [*http.Request]) and [ParseCredentialRequestResponseBody] (for
// [io.Reader]).
func ParseCredentialRequestResponseBytes(data []byte) (par *ParsedCredentialAssertionData, err error) {
	var car CredentialAssertionResponse

	if err = decodeBytes(data, &car); err != nil {
		return nil, ErrBadRequest.WithDetails("Parse error for Assertion").WithInfo(err.Error()).WithError(err)
	}

	return car.Parse()
}

// Parse validates and parses the [CredentialAssertionResponse] into a [ParsedCredentialAssertionData]. Most
// implementations should use [ParseCredentialRequestResponse], [ParseCredentialRequestResponseBody], or
// [ParseCredentialRequestResponseBytes] instead of calling this method directly.
func (car CredentialAssertionResponse) Parse() (par *ParsedCredentialAssertionData, err error) {
	if car.ID == "" {
		return nil, ErrBadRequest.WithDetails("CredentialAssertionResponse with ID missing")
	}

	if _, err = base64.RawURLEncoding.DecodeString(car.ID); err != nil {
		return nil, ErrBadRequest.WithDetails("CredentialAssertionResponse with ID not base64url encoded").WithError(err)
	}

	if car.Type != string(PublicKeyCredentialType) {
		return nil, ErrBadRequest.WithDetails("CredentialAssertionResponse with bad type")
	}

	var attachment AuthenticatorAttachment

	switch att := AuthenticatorAttachment(car.AuthenticatorAttachment); att {
	case Platform, CrossPlatform:
		attachment = att
	}

	par = &ParsedCredentialAssertionData{
		ParsedPublicKeyCredential{
			ParsedCredential{car.ID, car.Type}, car.RawID, car.ClientExtensionResults, attachment,
		},
		ParsedAssertionResponse{
			Signature:  car.AssertionResponse.Signature,
			UserHandle: car.AssertionResponse.UserHandle,
		},
		car,
	}

	// Step 5. Let JSONtext be the result of running UTF-8 decode on the value of cData.
	// We don't call it cData but this is Step 5 in the spec.
	if err = json.Unmarshal(car.AssertionResponse.ClientDataJSON, &par.Response.CollectedClientData); err != nil {
		return nil, err
	}

	if err = par.Response.AuthenticatorData.Unmarshal(car.AssertionResponse.AuthenticatorData); err != nil {
		return nil, ErrParsingData.WithDetails("Error unmarshalling auth data").WithError(err)
	}

	return par, nil
}

// Verify the remaining elements of the assertion data by following the steps outlined in the referenced specification
// documentation. It's important to note that the credentialBytes field is the CBOR representation of the credential.
//
// Specification: §7.2 Verifying an Authentication Assertion (https://www.w3.org/TR/webauthn/#sctn-verifying-assertion)
func (p *ParsedCredentialAssertionData) Verify(storedChallenge string, relyingPartyID, appID string, rpOrigins, rpTopOrigins []string, rpTopOriginsVerify TopOriginVerificationMode, allowCrossOrigin, verifyUser, verifyUserPresence bool, credentialBytes []byte) error {
	// Steps 4 through 6 in verifying the assertion data (https://www.w3.org/TR/webauthn/#verifying-assertion) are
	// "assertive" steps, i.e. "Let JSONtext be the result of running UTF-8 decode on the value of cData."
	// We handle these steps in part as we verify but also beforehand
	//
	// Handle steps 7 through 10 of assertion by verifying stored data against the Collected Client Data
	// returned by the authenticator.
	validError := p.Response.CollectedClientData.Verify(storedChallenge, AssertCeremony, rpOrigins, rpTopOrigins, rpTopOriginsVerify, allowCrossOrigin)
	if validError != nil {
		return validError
	}

	// Begin Step 11. Verify that the rpIdHash in authData is the SHA-256 hash of the RP ID expected by the RP.
	rpIDHash := sha256.Sum256([]byte(relyingPartyID))

	var appIDHash [32]byte
	if appID != "" {
		appIDHash = sha256.Sum256([]byte(appID))
	}

	// Handle steps 11 through 14, verifying the authenticator data.
	validError = p.Response.AuthenticatorData.Verify(rpIDHash[:], appIDHash[:], verifyUser, verifyUserPresence)
	if validError != nil {
		return validError
	}

	// Step 15. Let hash be the result of computing a hash over the cData using SHA-256.
	clientDataHash := sha256.Sum256(p.Raw.AssertionResponse.ClientDataJSON)

	// Step 16. Using the credential public key looked up in step 3, verify that sig is
	// a valid signature over the binary concatenation of authData and hash.

	sigData := append(p.Raw.AssertionResponse.AuthenticatorData, clientDataHash[:]...) //nolint:gocritic // This is intentional.

	var (
		key any
		err error
	)

	// If the Session Data does not contain the appID extension or it wasn't reported as used by the Client/RP then we
	// use the standard CTAP2 public key parser.
	if appID == "" {
		key, err = webauthncose.ParsePublicKey(credentialBytes)
	} else {
		key, err = webauthncose.ParseFIDOPublicKey(credentialBytes)
	}

	if err != nil {
		return ErrAssertionSignature.WithDetails(fmt.Sprintf("Error parsing the assertion public key: %+v", err)).WithError(err)
	}

	valid, err := webauthncose.VerifySignature(key, sigData, p.Response.Signature)
	if !valid || err != nil {
		return ErrAssertionSignature.WithDetails(fmt.Sprintf("Error validating the assertion signature: %+v", err)).WithError(err)
	}

	return nil
}
