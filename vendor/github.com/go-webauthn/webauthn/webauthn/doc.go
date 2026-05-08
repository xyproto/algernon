// Package webauthn contains the API functionality of the library. After creating and configuring a webauthn object,
// users can call the object to create and validate web authentication credentials.
//
// This documentation section highlights key functions within the library which are recommended and often have
// examples attached. Functions which are discouraged due to their lack of functionality are expressly not documented
// here, and you're on your own with these functions. Generally speaking, if the function is not documented here, it is
// either used by another function documented here, and it hides one of the arguments or return values, or it is lower
// level logic only intended for advanced use cases.
//
// The [New] function is a key function in creating a new instance of a WebAuthn Relying Party which is required to
// perform most actions.
//
// To start the credential creation ceremony, the [WebAuthn.BeginMediatedRegistration] or [WebAuthn.BeginRegistration]
// functions are used which returns [*SessionData] and a [*protocol.CredentialCreation] struct which can be easily
// serialized as JSON for the frontend library/logic. The [*SessionData] must be saved in a way which allows the
// implementer to restore it later. This [*SessionData] should be safely anchored to a user agent without allowing the
// user agent to modify the contents (i.e. opaque session cookie).
//
// To finish the credential creation ceremony, the [WebAuthn.FinishRegistration] function can be used. This function
// requires a [*http.Request] and performs all the necessary and requested validations. If you have other requirements,
// you can use [protocol.ParseCredentialCreationResponseBody] or [protocol.ParseCredentialCreationResponseBytes] which
// require an [io.Reader] or byte array respectively, then use [WebAuthn.CreateCredential] to
// perform validations against the [*protocol.ParsedCredentialCreationData] and saved [*SessionData] and finalize the
// process. For complete customizability, just produce the [*protocol.ParsedCredentialCreationData] with a custom parser
// and provide it to [WebAuthn.CreateCredential].
//
// To start a Passkey login ceremony, the [WebAuthn.BeginDiscoverableMediatedLogin] or [WebAuthn.BeginDiscoverableLogin]
// functions are used which returns [*SessionData] and a [*protocol.CredentialAssertion] struct which can easily be
// serialized as JSON for the frontend library/logic. The [*SessionData] should be safely handled as previously described.
//
// To finish a Passkey login ceremony, the [WebAuthn.FinishPasskeyLogin] function can be used. This function requires a
// [*http.Request] and performs all the necessary validations. If you have other requirements, you can use the
// [protocol.ParseCredentialRequestResponseBody] or [protocol.ParseCredentialRequestResponseBytes] which require an
// [io.Reader] or byte array respectively, then use [WebAuthn.ValidatePasskeyLogin] to perform validations against the
// [*protocol.ParsedCredentialAssertionData] and saved [*SessionData] and finalize the process. For complete customizabilty,
// just produce the [protocol.ParsedCredentialAssertionData] with a custom parser and provide it to
// [WebAuthn.ValidatePasskeyLogin].
//
// To start a Multi-Factor login ceremony, the [WebAuthn.BeginMediatedLogin] or [WebAuthn.BeginLogin]
// functions are used which returns [SessionData] and a [*protocol.CredentialAssertion] struct which can easily be
// serialized as JSON for the frontend library/logic. The [*SessionData] should be safely handled as previously described.
//
// To finish a Multi-Factor login ceremony, the [WebAuthn.FinishLogin] function can be used. This function requires a
// [*http.Request] and performs all the necessary validations. If you have other requirements, you can use the
// [protocol.ParseCredentialRequestResponseBody] or [protocol.ParseCredentialRequestResponseBytes] which require an
// [io.Reader] or byte array respectively, then use [WebAuthn.ValidateLogin] to perform validations against the
// [*protocol.ParsedCredentialAssertionData] and saved [*SessionData] and finalize the process. For complete
// customizabilty, just produce the [protocol.ParsedCredentialAssertionData] with a custom parser and provide it to
// [WebAuthn.ValidateLogin].
//
// # Relying Party Usage
//
// This library hadnles the relying party server-side concerns. The browser or other user agent is responsible for
// handling the JSON responses from this library and translating them for the WebAUthn API appropriately. There are two
// primary ways to handle this other than doing so manually:
//
//   1. Using a client side library like [@simplewebauthn/browser].
//   2. Some browsers support the [parseCreationOptionsFromJSON] static method on the WebAuthn object.
//
// [parseCreationOptionsFromJSON]: https://developer.mozilla.org/en-US/docs/Web/API/PublicKeyCredential/parseCreationOptionsFromJSON_static
// [@simplewebauthn/browser]: https://simplewebauthn.dev/docs/packages/browser
//
// # Storage
//
// This section describes how a Relying Party should persist the state produced by the library: the [Credential]
// records returned from registration (which must survive for the lifetime of the credential) and the
// [SessionData] records exchanged between the Begin and Finish/Validate calls of each ceremony (which need only
// live long enough to span the ceremony).
//
// Guidance here assumes PostgreSQL as the backing store; the same shape translates to other SQL engines but the
// column types given below are written against PostgreSQL.
//
// Two persistence shapes are supported for the [Credential] struct and the first is strongly recommended:
//
//  1. Explicit fields (recommended). Map each field of the struct (and for [Credential] the nested
//     [Authenticator] and [CredentialAttestation] fields) to its own column, using native types (BYTEA for raw
//     bytes, BOOLEAN for each flag, TIMESTAMPTZ for time values, etc.). This gives the database a typed,
//     queryable view of each record, allows per-field constraints and indexes, and lets an operator audit or
//     migrate individual values without having to decode an opaque blob.
//
//  2. Opaque serialized value. Serialize the whole struct into a single BYTEA (or JSONB) column. Both
//     encoding/json and MessagePack are supported via the struct tags on every field (the `msg:` tags drive the
//     msgp-generated code in *_gen.go, and the `json:` tags drive encoding/json); either encoding will
//     round-trip a [Credential]. Prefer this only when the explicit-field approach genuinely
//     does not fit; you lose the ability to query, index, or update individual fields in the database.
//
// One persistence shape is supported for the [SessionData] struct which is to store it as bytes via encoding/json or
// using MessagePack as bytes in whatever storage system you're using for user sessions. This data MUST be definitively
// anchored to a user's active session, and it must be restored between the ceremony steps.
//
// Regardless of which shape is chosen, the following values MUST be persisted as their own columns so records
// can be located and scoped correctly without first decoding attestation or key material. The User Handle in
// particular is per-user state (one value shared by every credential that user owns) and MUST NOT be stored on
// the credential row; store it once per user on a separate table (`webauthn_users` in the example below) and
// link credentials to that row via the application user identifier.
//
// On each [Credential] row:
//
//   - Credential ID; [Credential.ID], the identifier returned by the authenticator and echoed in every
//     assertion. This is the primary lookup key at login.
//   - Relying Party ID; the RP ID the credential was registered against. Credentials must be partitioned by
//     RP ID and the stored value must match the RP ID in effect at authentication time.
//   - Application user identifier; your application's own unique user id (the primary key used elsewhere in
//     your schema to reference the user). This is what ties a credential back to the user record and,
//     transitively via `webauthn_users`, to the User Handle.
//
// On a separate per-user row (`webauthn_users` or equivalent), keyed uniquely by (RP ID, application user id)
// and also uniquely by (RP ID, User Handle):
//
//   - Relying Party ID; same scoping rules as above; a user may have distinct User Handles under different
//     RP IDs, so the RP ID must be part of both unique keys on this table.
//   - Application user identifier; the same value stored on each of that user's credential rows; this is
//     the join column between `webauthn_users` and `webauthn_credentials`.
//   - User Handle; the opaque per-user byte sequence returned by [User.WebAuthnID], equivalently
//     [SessionData.UserID]. This is the value exchanged with the authenticator and is what
//     discoverable-credential flows return at login. It MUST be stable for the lifetime of the account and
//     MUST be the same across every credential that user owns; storing it once, per user, is what enforces
//     that. It is NOT the same as the application user identifier: the User Handle is an opaque WebAuthn
//     value emitted to authenticators, whereas the application user identifier is your schema's primary
//     key for the user. Keeping the two as separate columns lets you resolve from either direction.
//
// A minimal PostgreSQL schema covering the above plus the remaining [Credential], [Authenticator], and
// [CredentialAttestation] fields is shown below.
//
// Example users table:
//
//	CREATE TABLE webauthn_users (
//	    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
//	    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
//	    rpid        VARCHAR(512) NOT NULL, -- Relying Party ID
//	    user_id     UUID         NOT NULL, -- Application-side unique user id (FK to your users table)
//	    handle      BYTEA        NOT NULL  -- User.WebAuthnID (WebAuthn User Handle); stable per (rpid, user_id)
//	);
//
//	CREATE UNIQUE INDEX webauthn_users_user_id_key ON webauthn_users (rpid, user_id);
//	CREATE UNIQUE INDEX webauthn_users_handle_key  ON webauthn_users (rpid, handle);
//
// Example credentials table:
//
//	CREATE TABLE webauthn_credentials (
//	    id                       UUID         PRIMARY KEY DEFAULT uuidv7(),
//	    created_at               TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
//	    last_used_at             TIMESTAMPTZ  NULL,
//	    rpid                     VARCHAR(512) NOT NULL, -- Relying Party ID
//	    user_id                  UUID         NOT NULL, -- Application-side unique user id
//	    kid                      BYTEA        NOT NULL, -- Credential.ID
//	    aaguid                   BYTEA        NULL, -- Authenticator.AAGUID
//	    public_key               BYTEA        NOT NULL, -- Credential.PublicKey (encrypt at rest)
//	    attestation_type         VARCHAR(32)  NOT NULL, -- CredentialAttestation.AttestationType
//	    attestation_format       VARCHAR(32)  NOT NULL, -- CredentialAttestation.AttestationFormat
//	    attestation              BYTEA        NULL DEFAULT NULL, -- CredentialAttestation serialized as Message Pack or JSON (encrypt at rest)
//	    transport                VARCHAR(64)  NOT NULL DEFAULT '', -- Credential.Transport serialized as a comma-separated value
//	    sign_count               BIGINT       NOT NULL DEFAULT 0, -- Authenticator.SignCount
//	    clone_warning            BOOLEAN      NOT NULL DEFAULT FALSE, -- Authenticator.CloneWarning
//	    attachment               VARCHAR(64)  NOT NULL DEFAULT '', -- Authenticator.Attachment
//	    flags                    BYTEA        NOT NULL, -- Value of Flags.ProtocolValue (a single octet), restored with NewCredentialFlags, could also be SMALLINT
//	    present                  BOOLEAN      NOT NULL DEFAULT FALSE, -- Flags.UserPresent, optionally stored so you can either display it to the user or for filtering credentials
//	    verified                 BOOLEAN      NOT NULL DEFAULT FALSE, -- Flags.UserVerified, optionally stored so you can either display it to the user or for filtering credentials
//	    backup_eligible          BOOLEAN      NOT NULL DEFAULT FALSE, -- Flags.BackupEligible, optionally stored so you can either display it to the user or for filtering credentials
//	    backup_state             BOOLEAN      NOT NULL DEFAULT FALSE -- Flags.BackupState, optionally stored so you can either display it to the user or for filtering credentials
//	);
//
//	CREATE UNIQUE INDEX webauthn_credentials_kid_key ON webauthn_credentials (rpid, kid);
//	CREATE        INDEX webauthn_credentials_user_id ON webauthn_credentials (rpid, user_id);
//
// With that shape, the two login lookup paths resolve as:
//
//   - Credential-ID-first (allowCredentials flows): match `webauthn_credentials.kid` to the credential ID
//     returned by the authenticator, then optionally join `webauthn_users` on (rpid, user_id) to compare the
//     authenticator-supplied User Handle against the stored one.
//   - User-Handle-first (discoverable / passkey flows): match `webauthn_users.handle` under the current
//     RP ID to resolve the application user id, then load that user's credentials from
//     `webauthn_credentials`.
//
// Fields that change across assertions; [Authenticator.SignCount], [Authenticator.CloneWarning], and
// [CredentialFlags.BackupState] when [CredentialFlags.BackupEligible] is true MUST be written back to storage
// on every successful FinishLogin / ValidateLogin so the next ceremony observes the current values.
//
// For [SessionData] stored in a database (rather than a server-side session store), use the same persistence
// shapes described above. The User Handle on a [SessionData] row is per-session ceremony state rather than
// per-user state, so it is fine to keep [SessionData.UserID] on the session row itself; the per-user-table
// rule applies to [Credential] storage, not to [SessionData]. Additionally index the challenge (unique) and
// the expiry timestamp so sessions can be looked up by challenge at Finish time and expired rows reaped
// cheaply. Stored sessions must only be consumed by a Finish call operating under the same RP ID.
package webauthn
