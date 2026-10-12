// Package webauthn provides Lua functions for passwordless authentication using WebAuthn/FIDO2
package webauthn

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/pinterface/v2"
)

// user implements webauthn.User backed by the Algernon userstate and credential store
type user struct {
	id          []byte
	name        string
	credentials []webauthn.Credential
}

func (u *user) WebAuthnID() []byte                         { return u.id }
func (u *user) WebAuthnName() string                       { return u.name }
func (u *user) WebAuthnDisplayName() string                { return u.name }
func (u *user) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

const (
	credentialsField = "webauthn_creds"
	sessionField     = "webauthn_session"
	legacyCredStore  = "webauthn:creds"
)

// credStore persists WebAuthn credentials as JSON, in a field of each user
type credStore struct {
	userstate pinterface.IUserState
}

// loadUser returns a user with stored credentials, or a new user if none exist
func (cs *credStore) loadUser(username string) *user {
	u := &user{id: []byte(username), name: username}
	raw, err := cs.userstate.Users().Get(username, credentialsField)
	if err != nil || raw == "" {
		raw = cs.migrateLegacy(username)
	}
	if raw == "" {
		return u
	}
	var creds []webauthn.Credential
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return u
	}
	u.credentials = creds
	return u
}

// migrateLegacy moves the credentials of the given user from the legacy store
// to the user, and returns them
func (cs *credStore) migrateLegacy(username string) string {
	kv, err := cs.userstate.Creator().NewKeyValue(legacyCredStore)
	if err != nil {
		return ""
	}
	raw, err := kv.Get(username)
	if err != nil || raw == "" {
		return ""
	}
	if err := cs.userstate.Users().Set(username, credentialsField, raw); err == nil {
		kv.Del(username)
	}
	return raw
}

// addCredential appends a credential and persists the updated list
func (cs *credStore) addCredential(username string, cred *webauthn.Credential) error {
	u := cs.loadUser(username)
	u.credentials = append(u.credentials, *cred)
	b, err := json.Marshal(u.credentials)
	if err != nil {
		return err
	}
	return cs.userstate.Users().Set(username, credentialsField, string(b))
}

// sessionStore persists WebAuthn challenge session data between begin/finish calls
type sessionStore struct {
	userstate pinterface.IUserState
}

func (ss *sessionStore) save(username string, session *webauthn.SessionData) error {
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return ss.userstate.Users().Set(username, sessionField, string(b))
}

func (ss *sessionStore) load(username string) (*webauthn.SessionData, error) {
	raw, err := ss.userstate.Users().Get(username, sessionField)
	if err != nil {
		return nil, err
	}
	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (ss *sessionStore) delete(username string) {
	ss.userstate.Users().DelKey(username, sessionField)
}

// Load makes WebAuthn functions available to Lua scripts.
// The rpID and rpOrigin are derived from the current request if not configured.
func Load(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate pinterface.IUserState) {
	cs := &credStore{userstate: userstate}
	ss := &sessionStore{userstate: userstate}

	// Passkeys may only be added to an existing account, by that logged in user
	isAccountOwner := func(username string) bool {
		if username == "" || !userstate.HasUser(username) {
			return false
		}
		cookieUser, err := userstate.UsernameCookie(req)
		return err == nil && cookieUser == username && userstate.IsLoggedIn(username)
	}

	// Derive the relying party config from the request
	newWebAuthn := func() (*webauthn.WebAuthn, error) {
		scheme := "https"
		if req.TLS == nil {
			scheme = "http"
		}
		host := req.Host
		rpID, _, err := net.SplitHostPort(host)
		if err != nil {
			rpID = host // no port present
		}
		origin := scheme + "://" + host
		return webauthn.New(&webauthn.Config{
			RPDisplayName: rpID,
			RPID:          rpID,
			RPOrigins:     []string{origin},
		})
	}

	// Begin a WebAuthn registration ceremony.
	// Takes a username, writes JSON options to the response.
	L.SetGlobal("WebAuthnBeginRegister", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		if !isAccountOwner(username) {
			logrus.Warn("WebAuthn: only a logged in user can register a passkey for their own account, not for ", username)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		wa, err := newWebAuthn()
		if err != nil {
			logrus.Error("WebAuthn config: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		u := cs.loadUser(username)
		creation, session, err := wa.BeginRegistration(u)
		if err != nil {
			logrus.Error("WebAuthn begin registration: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		if err := ss.save(username, session); err != nil {
			logrus.Error("WebAuthn save session: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(creation); err != nil {
			logrus.Error("WebAuthn encode options: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Finish a WebAuthn registration ceremony.
	// Takes a username, reads the attestation response from the request body.
	// Returns true if the credential was stored.
	L.SetGlobal("WebAuthnFinishRegister", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		if !isAccountOwner(username) {
			logrus.Warn("WebAuthn: only a logged in user can register a passkey for their own account, not for ", username)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		wa, err := newWebAuthn()
		if err != nil {
			logrus.Error("WebAuthn config: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		session, err := ss.load(username)
		if err != nil {
			logrus.Error("WebAuthn load session: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		u := cs.loadUser(username)
		cred, err := wa.FinishRegistration(u, *session, req)
		if err != nil {
			logrus.Error("WebAuthn finish registration: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		ss.delete(username)
		if err := cs.addCredential(username, cred); err != nil {
			logrus.Error("WebAuthn store credential: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Begin a WebAuthn login ceremony.
	// Takes a username, writes JSON options to the response.
	L.SetGlobal("WebAuthnBeginLogin", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		if !userstate.HasUser(username) {
			logrus.Warn("WebAuthn: no such user: ", username)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		wa, err := newWebAuthn()
		if err != nil {
			logrus.Error("WebAuthn config: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		u := cs.loadUser(username)
		if len(u.credentials) == 0 {
			logrus.Warn("WebAuthn: no credentials for ", username)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		assertion, session, err := wa.BeginLogin(u)
		if err != nil {
			logrus.Error("WebAuthn begin login: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		if err := ss.save(username, session); err != nil {
			logrus.Error("WebAuthn save session: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(assertion); err != nil {
			logrus.Error("WebAuthn encode options: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Finish a WebAuthn login ceremony.
	// Takes a username, reads the assertion response from the request body.
	// Returns true if authentication succeeded. Also logs the user in.
	L.SetGlobal("WebAuthnFinishLogin", L.NewFunction(func(L *lua.LState) int {
		username := L.ToString(1)
		if !userstate.HasUser(username) {
			logrus.Warn("WebAuthn: no such user: ", username)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		wa, err := newWebAuthn()
		if err != nil {
			logrus.Error("WebAuthn config: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		session, err := ss.load(username)
		if err != nil {
			logrus.Error("WebAuthn load session: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		u := cs.loadUser(username)
		_, err = wa.FinishLogin(u, *session, req)
		if err != nil {
			logrus.Error("WebAuthn finish login: ", err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		ss.delete(username)
		// Log the user in via the standard session mechanism
		if loginErr := userstate.Login(w, username); loginErr != nil {
			logrus.Error("WebAuthn login session: ", loginErr)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))
}
