package webauthn

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/protocol"
)

//go:generate msgp

//msgp:replace protocol.AuthenticatorTransport with:string
//msgp:shim CredentialFlags as:byte using:(CredentialFlags).MsgpByte/CredentialFlagsFromMsgpByte
//msgp:clearomitted

// NewCredential returns a [*Credential] from a successfully validated registration response. The returned Credential
// includes a populated [CredentialAttestation] containing the raw attestation data needed for future verification;
// see the [CredentialAttestation] documentation for why these values must be persisted.
func NewCredential(clientDataHash []byte, c *protocol.ParsedCredentialCreationData) (credential *Credential, err error) {
	credential = &Credential{
		ID:                c.Response.AttestationObject.AuthData.AttData.CredentialID,
		PublicKey:         c.Response.AttestationObject.AuthData.AttData.CredentialPublicKey,
		AttestationType:   c.Response.AttestationObject.Type,
		AttestationFormat: c.Response.AttestationObject.Format,
		Transport:         c.Response.Transports,
		Flags:             NewCredentialFlags(c.Response.AttestationObject.AuthData.Flags),
		Authenticator: Authenticator{
			AAGUID:     c.Response.AttestationObject.AuthData.AttData.AAGUID,
			SignCount:  c.Response.AttestationObject.AuthData.Counter,
			Attachment: c.AuthenticatorAttachment,
		},
		Attestation: CredentialAttestation{
			ClientDataJSON:     c.Raw.AttestationResponse.ClientDataJSON,
			ClientDataHash:     clientDataHash,
			AuthenticatorData:  c.Raw.AttestationResponse.AuthenticatorData,
			PublicKeyAlgorithm: c.Raw.AttestationResponse.PublicKeyAlgorithm,
			Object:             c.Raw.AttestationResponse.AttestationObject,
		},
	}

	return credential, nil
}

// Credential contains all needed information about a WebAuthn credential for storage. This struct is effectively the
// Credential Record as described in the specification.
//
// Provided this data structure is preserved properly, a Credential can be verified against the FIDO Metadata Service
// at a later date using the [Credential.Verify] method with a [metadata.Provider].
//
// It is strongly recommended for the best security that a [Credential] is encrypted at rest with the exception of the
// ID and the value you use to lookup the user. This prevents a person with access to the database being able to
// compromise privacy by being able to view this data, as well as prevents them being able to compromise security by
// adding or modifying a Credential without them also having access to the encryption key.
//
// For consolidated persistence guidance; recommended schema shape, required lookup columns, and which fields
// must be written back on every successful FinishLogin / ValidateLogin; see the [Storage] section of the
// [github.com/go-webauthn/webauthn/webauthn] package documentation.
//
// See: §4. Terminology: Credential Record (https://www.w3.org/TR/webauthn-3/#credential-record)
//
// [Storage]: https://pkg.go.dev/github.com/go-webauthn/webauthn/webauthn#hdr-Storage
type Credential struct {
	// The ID is the ID of the public key credential source. Described by the Credential Record 'id' field.
	ID []byte `json:"id" msg:"id"`

	// The credential public key of the public key credential source. Described by the Credential Record 'publicKey'
	// field.
	PublicKey []byte `json:"publicKey" msg:"pk"`

	// AttestationType is the attestation type as conveyed by the authenticator during the registration ceremonyl
	// one of the values defined by [metadata.AuthenticatorAttestationType] ("basic_full", "basic_surrogate",
	// "attca", "anonca", "ecdaa", "none"). Prior releases incorrectly stored the attestation FORMAT here; see the
	// custom [Credential.UnmarshalJSON] for the backward-compatibility migration applied when decoding such
	// records.
	AttestationType string `json:"attestationType,omitempty" msg:"atttype,omitempty"`

	// AttestationFormat is the attestation statement format identifier ("packed", "tpm", "android-key",
	// "android-safetynet", "fido-u2f", "apple", "compound", "none"); see §8 of the WebAuthn specification and
	// the AttestationFormat constants in the protocol package.
	AttestationFormat string `json:"attestationFormat,omitempty" msg:"attfmt,omitempty"`

	// Transport types the authenticator supports. Described by the Credential Record 'transports' field.
	Transport []protocol.AuthenticatorTransport `json:"transport,omitempty" msg:"t,omitempty"`

	// Flags represent the commonly stored flags.
	Flags CredentialFlags `json:"flags" msg:"flg"`

	// The Authenticator information for a given Credential.
	Authenticator Authenticator `json:"authenticator" msg:"a"`

	// The attestation values that can be used to validate this Credential via the MDS3 at a later date.
	Attestation CredentialAttestation `json:"attestation" msg:"att"`
}

// UnmarshalJSON decodes a [Credential] from JSON, applying a backward-compatibility migration for records produced
// by earlier versions of this library: if the decoded record has no AttestationFormat and the AttestationType value
// is a recognised attestation FORMAT identifier (i.e. "packed", "tpm", "none"), the value is moved to
// AttestationFormat and AttestationType is cleared so callers can re-derive the true attestation type by calling
// [Credential.Verify]. Records that already carry an AttestationFormat are untouched.
func (c *Credential) UnmarshalJSON(data []byte) error {
	type credentialAlias Credential

	var tmp credentialAlias

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	*c = Credential(tmp)

	if c.AttestationFormat == "" && protocol.IsAttestationFormatString(c.AttestationType) {
		c.AttestationFormat = c.AttestationType
		c.AttestationType = ""
	}

	return nil
}

// SignalUnknownCredential creates a struct that can easily be marshaled to JSON which indicates this is an unknown
// Credential.
func (c *Credential) SignalUnknownCredential(rpid string) *protocol.SignalUnknownCredential {
	return c.Descriptor().SignalUnknownCredential(rpid)
}

// Descriptor converts a [Credential] into a [protocol.CredentialDescriptor].
func (c *Credential) Descriptor() (descriptor protocol.CredentialDescriptor) {
	return protocol.CredentialDescriptor{
		Type:              protocol.PublicKeyCredentialType,
		CredentialID:      c.ID,
		Transport:         c.Transport,
		AttestationType:   c.AttestationType,
		AttestationFormat: c.AttestationFormat,
	}
}

// Verify re-runs the full attestation verification for this credential against the given [metadata.Provider]. The
// stored raw attestation bytes are re-parsed, the attestation signature is re-verified, and the authenticator is
// validated against the MDS via [protocol.AttestationObject.VerifyAttestation] (which internally dispatches
// [protocol.ValidateMetadata]). This is the canonical audit path and is at least as strong as the original
// registration-time verification; call it on a schedule (i.e. on login or periodically) to catch MDS status changes
// such as a newly-revoked authenticator model or a compromise advisory published after registration.
//
// Requirements:
//
//   - The mds argument must be a non-nil [metadata.Provider]; a nil provider returns an error.
//
//   - [CredentialAttestation.ClientDataJSON] must be preserved byte-for-byte; it is re-parsed for its collected
//     client data fields and re-hashed when [CredentialAttestation.ClientDataHash] is absent.
//
//   - [CredentialAttestation.Object] must be preserved byte-for-byte; it is the raw CBOR attestation object and
//     is decoded to recover the authenticator data, statement format, and statement for full re-verification.
//
//   - [Credential.PublicKey] must be populated with the CBOR-encoded COSE key as emitted by the authenticator at
//     registration. As an integrity check, Verify compares this value byte-for-byte against the credential public
//     key carried inside the attestation object and returns an error on mismatch.
//
//   - [CredentialAttestation.ClientDataHash] is optional; if empty it is recomputed as the SHA-256 of
//     ClientDataJSON.
//
//   - [Credential.Transport], [CredentialAttestation.AuthenticatorData], and [CredentialAttestation.PublicKeyAlgorithm]
//     are not read by the current Verify implementation (the authenticator data is re-derived from the attestation
//     object, and the top-level AuthenticatorData / PublicKeyAlgorithm convenience fields are informational). They
//     are still stored so future versions of this library, or alternative verification paths, can consume them;
//     see [CredentialAttestation] for why every field should be persisted.
//
// As a side-effect, a successful Verify call will populate [Credential.AttestationType] from the re-derived value
// when the field is empty (i.e. on a record migrated from a pre-split JSON layout by [Credential.UnmarshalJSON]);
// the next marshal of the Credential will then carry the correct attestation type. For this reason Verify uses a
// pointer receiver.
//
// See [CredentialAttestation] for guidance on persisting these raw values securely.
func (c *Credential) Verify(mds metadata.Provider) (err error) {
	if mds == nil {
		return fmt.Errorf("error verifying credential: the metadata provider must be provided but it's nil")
	}

	raw := c.toAuthenticatorAttestationResponse()

	var attestation *protocol.ParsedAttestationResponse

	if attestation, err = raw.Parse(); err != nil {
		return fmt.Errorf("error verifying credential: error parsing attestation: %w", err)
	}

	if !bytes.Equal(c.PublicKey, attestation.AttestationObject.AuthData.AttData.CredentialPublicKey) {
		return fmt.Errorf("error verifying credential: stored public key does not match the credential public key embedded in the attestation object")
	}

	clientDataHash := c.Attestation.ClientDataHash

	if len(clientDataHash) == 0 {
		sum := sha256.Sum256(c.Attestation.ClientDataJSON)

		clientDataHash = sum[:]
	}

	if err = attestation.AttestationObject.VerifyAttestation(clientDataHash, mds); err != nil {
		return fmt.Errorf("error verifying credential: error verifying attestation: %w", err)
	}

	if c.AttestationType == "" {
		c.AttestationType = attestation.AttestationObject.Type
	}

	return nil
}

// VerifyAttestationType is a cutdown version of Verify which only does the minimal verification to update the
// AttestationType if it's unset. For full verification use Verify.
func (c *Credential) VerifyAttestationType() (err error) {
	if c.AttestationType != "" {
		return nil
	}

	raw := c.toAuthenticatorAttestationResponse()

	var attestation *protocol.ParsedAttestationResponse

	if attestation, err = raw.Parse(); err != nil {
		return fmt.Errorf("error verifying credential: error parsing attestation: %w", err)
	}

	if !bytes.Equal(c.PublicKey, attestation.AttestationObject.AuthData.AttData.CredentialPublicKey) {
		return fmt.Errorf("error verifying credential: stored public key does not match the credential public key embedded in the attestation object")
	}

	clientDataHash := c.Attestation.ClientDataHash

	if len(clientDataHash) == 0 {
		sum := sha256.Sum256(c.Attestation.ClientDataJSON)

		clientDataHash = sum[:]
	}

	if err = attestation.AttestationObject.VerifyAttestation(clientDataHash, nil); err != nil {
		return fmt.Errorf("error verifying credential: error verifying attestation: %w", err)
	}

	c.AttestationType = attestation.AttestationObject.Type

	return nil
}

func (c *Credential) toAuthenticatorAttestationResponse() *protocol.AuthenticatorAttestationResponse {
	raw := &protocol.AuthenticatorAttestationResponse{
		AuthenticatorResponse: protocol.AuthenticatorResponse{
			ClientDataJSON: c.Attestation.ClientDataJSON,
		},
		Transports:         make([]string, len(c.Transport)),
		AuthenticatorData:  c.Attestation.AuthenticatorData,
		PublicKey:          c.PublicKey,
		PublicKeyAlgorithm: c.Attestation.PublicKeyAlgorithm,
		AttestationObject:  c.Attestation.Object,
	}

	for i, transport := range c.Transport {
		raw.Transports[i] = string(transport)
	}

	return raw
}

// Credentials is a decorator type which allows easily converting a [Credential] slice into a
// [protocol.CredentialDescriptor] slice by utilizing the [Credentials.CredentialDescriptors] method. This will be the
// type used globally for the library in a future release.
type Credentials []Credential

// CredentialDescriptors returns the [protocol.CredentialDescriptor] slice for this [Credentials] type.
func (c Credentials) CredentialDescriptors() (descriptors []protocol.CredentialDescriptor) {
	descriptors = make([]protocol.CredentialDescriptor, len(c))

	for i, credential := range c {
		descriptors[i] = credential.Descriptor()
	}

	return descriptors
}

// NewCredentialFlags is a utility function that is used to derive the [Credential]'s Flags field given a
// [protocol.AuthenticatorFlags]. This allows implementers to solely save the Raw field of the [CredentialFlags] to
// restore them appropriately for appropriate processing without concern that changes forced upon implementers by the
// W3C will introduce breaking changes.
func NewCredentialFlags(flags protocol.AuthenticatorFlags) CredentialFlags {
	return CredentialFlags{
		UserPresent:    flags.HasUserPresent(),
		UserVerified:   flags.HasUserVerified(),
		BackupEligible: flags.HasBackupEligible(),
		BackupState:    flags.HasBackupState(),
		raw:            flags,
	}
}

// CredentialFlagsFromMsgpByte reconstructs a [CredentialFlags] from the single-byte representation produced by
// [CredentialFlags.MsgpByte]. It is intended for use by the msgp-generated serialization layer; normal callers
// should prefer [NewCredentialFlags].
func CredentialFlagsFromMsgpByte(b byte) CredentialFlags {
	return NewCredentialFlags(protocol.AuthenticatorFlags(b))
}

// CredentialFlags contains the boolean flags derived from the authenticator data during registration or login.
// These flags indicate the state of user presence, user verification, and backup eligibility/state at the time
// the credential was used.
type CredentialFlags struct {
	// Flag UP indicates the users presence.
	UserPresent bool `json:"userPresent"`

	// Flag UV indicates the user performed verification.
	UserVerified bool `json:"userVerified"`

	// Flag BE indicates the credential is able to be backed up and/or sync'd between devices. This should NEVER change.
	BackupEligible bool `json:"backupEligible"`

	// Flag BS indicates the credential has been backed up and/or sync'd. This value can change but it's recommended
	// that RP's keep track of this value.
	BackupState bool `json:"backupState"`

	raw protocol.AuthenticatorFlags
}

// ProtocolValue returns the underlying [protocol.AuthenticatorFlags] provided this [CredentialFlags] was created using
// NewCredentialFlags.
func (f CredentialFlags) ProtocolValue() protocol.AuthenticatorFlags {
	return f.raw
}

// MsgpByte returns the [CredentialFlags] encoded as a single byte, equivalent to the raw
// [protocol.AuthenticatorFlags] value. It is intended for use by the msgp-generated serialization layer (see the
// //msgp:shim directive in this file); normal callers should prefer [CredentialFlags.ProtocolValue].
func (f CredentialFlags) MsgpByte() byte {
	return byte(f.raw)
}

// CredentialAttestation holds the raw attestation data from a registration ceremony. These values are intentionally
// stored in their original unparsed form rather than as parsed structures. This is critical because:
//
//   - It enables the [Credential] to be verified against the FIDO Metadata Service at a later date using
//     [Credential.Verify], even long after the registration ceremony has completed.
//   - The WebAuthn specification evolves over time, introducing new validation procedures. Preserving the raw data
//     ensures that credentials created today can be re-validated against future rules without requiring re-registration.
//   - Raw data serves as an auditable record of exactly what the authenticator and client provided during registration,
//     independent of how the library parsed it at that point in time.
//
// Implementers MUST persist all fields of this struct.
type CredentialAttestation struct {
	// ClientDataJSON is the raw JSON-encoded client data from the registration response. This is the verbatim value
	// provided by the client and is used to recompute the client data hash during later verification.
	ClientDataJSON []byte `json:"clientDataJSON,omitempty" msg:"cdj,omitempty"`

	// ClientDataHash is the SHA-256 hash of ClientDataJSON computed during registration verification. If empty,
	// [Credential.Verify] will recompute it from ClientDataJSON.
	ClientDataHash []byte `json:"clientDataHash,omitempty" msg:"cdh,omitempty"`

	// AuthenticatorData is the raw authenticator data from the registration response as provided in the
	// RegistrationResponseJSON. This is the unparsed byte representation that can be re-parsed for future validation.
	AuthenticatorData []byte `json:"authenticatorData,omitempty" msg:"data,omitempty"`

	// PublicKeyAlgorithm is the COSE algorithm identifier for the credential's public key.
	PublicKeyAlgorithm int64 `json:"publicKeyAlgorithm,omitempty" msg:"alg,omitempty"`

	// Object is the raw CBOR-encoded attestation object from the registration response. This contains the attestation
	// statement, format, and authenticator data needed by [Credential.Verify] to re-perform attestation verification.
	Object []byte `json:"object,omitempty" msg:"obj,omitempty"`
}
