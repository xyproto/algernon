package webauthn

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/protocol"
)

// New creates a new [WebAuthn] instance from the provided [Config]. The configuration is validated before the
// instance is returned.
func New(config *Config) (*WebAuthn, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf(errFmtConfigValidate, err)
	}

	return &WebAuthn{
		config,
	}, nil
}

// WebAuthn is the primary interface of this package. It provides methods to begin and finish both registration and
// login ceremonies. Create an instance using [New] and then call the appropriate Begin/Finish methods for your
// use case. See the package documentation for detailed ceremony flows.
type WebAuthn struct {
	Config *Config
}

// Config represents the Relying Party configuration for WebAuthn operations. At minimum, RPID and RPOrigins must
// be configured. The RPID should be the effective domain of the Relying Party (i.e. "example.com") and RPOrigins
// should contain the fully qualified origins that are permitted (i.e. "https://example.com").
type Config struct {
	// RPID configures the Relying Party Server ID. This should generally be the origin without a scheme and port.
	RPID string

	// RPDisplayName configures the display name for the Relying Party Server. This can be any string.
	RPDisplayName string

	// RPOrigins configures the list of Relying Party Server Origins that are permitted. The provided origins can either
	// be fully qualified origins or strings for simple string comparison. The strings are matched using canonical
	// origin matching semantics specifically if they start with 'http://' or 'https://' if the provided origin has a
	// case-insensitive equal scheme and host component they are equal, otherwise simple string comparison is utilized
	// to determine equality.
	RPOrigins []string

	// RPTopOrigins configures the list of Relying Party Server Top Origins that are permitted. The provided origins can
	// either be fully qualified origins or strings for simple string comparison. The strings are matched using
	// canonical origin matching semantics specifically if they start with 'http://' or 'https://' if the provided
	// origin has a case-insensitive equal scheme and host component they are equal, otherwise simple string comparison
	// is utilized to determine equality.
	RPTopOrigins []string

	// RPTopOriginVerificationMode determines the verification mode for the Top Origin value used in cross-origin
	// ceremonies. When the zero value ([protocol.TopOriginDefaultVerificationMode]) is provided, the config
	// validator coerces this field to [protocol.TopOriginExplicitVerificationMode]; i.e. any Top Origin supplied
	// by the client must appear in [Config.RPTopOrigins]. Set this field explicitly to
	// [protocol.TopOriginAutoVerificationMode] or [protocol.TopOriginImplicitVerificationMode] if you need
	// different matching semantics; there is no longer a mode that disables verification entirely.
	RPTopOriginVerificationMode protocol.TopOriginVerificationMode

	// RPAllowCrossOrigin determines whether the RP is allowed to be used in cross-origin contexts. This is disabled
	// by default.
	RPAllowCrossOrigin bool

	// AttestationPreference sets the default attestation conveyance preferences.
	AttestationPreference protocol.ConveyancePreference

	// AuthenticatorSelection sets the default authenticator selection options.
	AuthenticatorSelection protocol.AuthenticatorSelection

	// Debug enables various debug options.
	Debug bool

	// EncodeUserIDAsString ensures the user.id value during registrations is encoded as a raw UTF8 string. This is
	// useful when you only use printable ASCII characters for the random user.id but the browser library does not
	// decode the URL Safe Base64 data.
	EncodeUserIDAsString bool

	// Timeouts configures various timeouts.
	Timeouts TimeoutsConfig

	// MDS configures a FIDO Metadata Service provider for authenticator trust validation. When set, the library
	// validates attestation statements against known authenticator metadata including trust anchors, attestation
	// types, and authenticator status. Use the providers in [github.com/go-webauthn/webauthn/metadata/providers/memory]
	// or [github.com/go-webauthn/webauthn/metadata/providers/cached] to create a provider instance.
	MDS metadata.Provider

	// Filtering configures the filtering of authenticators based on their AAGUIDs. This is useful for enforcing
	// policy on the authenticators that are available to be registered with the Relying Party.
	Filtering *FilteringConfig

	validated bool
}

// FilteringConfig configures the filtering of authenticators based on their AAGUIDs. This is useful for enforcing
// policy on the authenticators that are available to be registered with the Relying Party.
type FilteringConfig struct {
	// ProhibitBackupEligibility if set will prohibit the use of authenticators with the backup eligible flag set.
	ProhibitBackupEligibility bool

	// PermittedAAGUIDs if set is used to filter authenticators by their AAGUID only allowing specific values. This
	// option is mutually exclusive with ProhibitedAAGUIDs and will never exclude a zero AAGUID. To prohibit the use
	// of Zero AAGUIDs, use [Config.MDS] or [FilteringConfig.ProhibitedAAGUIDs].
	PermittedAAGUIDs []uuid.UUID

	// ProhibitedAAGUIDs if set is used to filter authenticators by their AAGUID only prohibiting specific values. This
	// option is mutually exclusive with PermittedAAGUIDs.
	ProhibitedAAGUIDs []uuid.UUID
}

// TimeoutsConfig configures the timeout durations for both login and registration ceremonies. These values are sent
// to the client as the timeout field in the credential request/creation options and optionally enforced server-side.
type TimeoutsConfig struct {
	Login        TimeoutConfig
	Registration TimeoutConfig
}

// TimeoutConfig configures timeout behavior for a specific WebAuthn ceremony (registration or login).
type TimeoutConfig struct {
	// Enforce the timeouts at the Relying Party / Server. This means if enabled and the user takes too long that even
	// if the browser does not enforce the timeout the Relying Party / Server will.
	Enforce bool

	// Timeout is the timeout for logins/registrations when the UserVerificationRequirement is set to anything other
	// than discouraged.
	Timeout time.Duration

	// TimeoutUVD is the timeout for logins/registrations when the UserVerificationRequirement is set to discouraged.
	TimeoutUVD time.Duration
}

// Validate that the config flags in Config are properly set.
func (config *Config) validate() (err error) {
	if config.validated {
		return nil
	}

	if len(config.RPID) != 0 {
		if err = protocol.ValidateRPID(config.RPID); err != nil {
			return fmt.Errorf(errFmtFieldNotValidDomainString, "RPID", err)
		}
	}

	defaultTimeoutConfig := defaultTimeout
	defaultTimeoutUVDConfig := defaultTimeoutUVD

	if config.Timeouts.Login.Timeout.Milliseconds() == 0 {
		config.Timeouts.Login.Timeout = defaultTimeoutConfig
	}

	if config.Timeouts.Login.TimeoutUVD.Milliseconds() == 0 {
		config.Timeouts.Login.TimeoutUVD = defaultTimeoutUVDConfig
	}

	if config.Timeouts.Registration.Timeout.Milliseconds() == 0 {
		config.Timeouts.Registration.Timeout = defaultTimeoutConfig
	}

	if config.Timeouts.Registration.TimeoutUVD.Milliseconds() == 0 {
		config.Timeouts.Registration.TimeoutUVD = defaultTimeoutUVDConfig
	}

	if len(config.RPOrigins) == 0 {
		return fmt.Errorf("must provide at least one value to the 'RPOrigins' field")
	}

	if config.RPTopOriginVerificationMode == protocol.TopOriginDefaultVerificationMode {
		config.RPTopOriginVerificationMode = protocol.TopOriginExplicitVerificationMode
	}

	if config.Filtering != nil {
		if len(config.Filtering.PermittedAAGUIDs) > 0 && len(config.Filtering.ProhibitedAAGUIDs) > 0 {
			return fmt.Errorf("cannot set both 'PermittedAAGUIDs' and 'ProhibitedAAGUIDs' in the filtering config")
		}
	}

	config.validated = true

	return nil
}

// GetRPID returns the configured Relying Party ID.
func (c *Config) GetRPID() string {
	return c.RPID
}

// GetOrigins returns the configured Relying Party Origins.
func (c *Config) GetOrigins() []string {
	return c.RPOrigins
}

// GetTopOrigins returns the configured Relying Party Top Origins.
func (c *Config) GetTopOrigins() []string {
	return c.RPTopOrigins
}

// GetTopOriginVerificationMode returns the configured Top Origin verification mode.
func (c *Config) GetTopOriginVerificationMode() protocol.TopOriginVerificationMode {
	return c.RPTopOriginVerificationMode
}

// GetMetaDataProvider returns the configured FIDO Metadata Service provider.
func (c *Config) GetMetaDataProvider() metadata.Provider {
	return c.MDS
}

// ConfigProvider is an interface that provides access to the WebAuthn [Config] values. This is useful for
// implementations that wish to provide configuration from alternative sources.
type ConfigProvider interface {
	GetRPID() string
	GetOrigins() []string
	GetTopOrigins() []string
	GetTopOriginVerificationMode() protocol.TopOriginVerificationMode
	GetMetaDataProvider() metadata.Provider
}

// User is an interface with the Relying Party's User entry and provides the fields and methods needed for WebAuthn
// registration operations.
type User interface {
	// WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
	// size of 64 bytes, and is not meant to be displayed to the user.
	//
	// To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
	// member, not the displayName nor name members. See Section 6.1 of [RFC8266].
	//
	// It's recommended this value is completely random and uses the entire 64 bytes.
	//
	// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
	WebAuthnID() []byte

	// WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name
	// for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD
	// let the user choose this, and SHOULD NOT restrict the choice more than necessary.
	//
	// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
	WebAuthnName() string

	// WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
	// name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
	// SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
	//
	// Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
	WebAuthnDisplayName() string

	// WebAuthnCredentials provides the slice of [Credential] objects owned by the user. This generally should be all
	// the [Credential] objects owned by the user regardless of which flow is being used.
	WebAuthnCredentials() []Credential
}
