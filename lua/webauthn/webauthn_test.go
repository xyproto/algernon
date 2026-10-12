package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/xyproto/gopher-lua"
	bolt "github.com/xyproto/permissionbolt/v2"
	sqlite "github.com/xyproto/permissionsqlite_noc"
	"github.com/xyproto/pinterface/v2"
)

const (
	testHost   = "localhost"
	testOrigin = "http://" + testHost
)

// authenticator is a minimal software passkey, using the "none" attestation
type authenticator struct {
	key       *ecdsa.PrivateKey
	credID    []byte
	signCount uint32
}

func newAuthenticator(t *testing.T) *authenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	credID := make([]byte, 16)
	rand.Read(credID)
	return &authenticator{key: key, credID: credID}
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func cborHead(major byte, n int) []byte {
	switch {
	case n < 24:
		return []byte{major<<5 | byte(n)}
	case n < 256:
		return []byte{major<<5 | 24, byte(n)}
	default:
		return []byte{major<<5 | 25, byte(n >> 8), byte(n)}
	}
}

func cborBytes(b []byte) []byte { return append(cborHead(2, len(b)), b...) }

func cborText(s string) []byte { return append(cborHead(3, len(s)), s...) }

func (a *authenticator) coseKey(t *testing.T) []byte {
	pub, err := a.key.PublicKey.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	raw := pub.Bytes() // 0x04 || x || y
	k := cborHead(5, 5)
	k = append(append(k, cborHead(0, 1)...), cborHead(0, 2)...) // kty: EC2
	k = append(append(k, cborHead(0, 3)...), cborHead(1, 6)...) // alg: ES256 (-7)
	k = append(append(k, cborHead(1, 0)...), cborHead(0, 1)...) // crv: P-256
	k = append(append(k, cborHead(1, 1)...), cborBytes(raw[1:33])...)
	k = append(append(k, cborHead(1, 2)...), cborBytes(raw[33:])...)
	return k
}

func (a *authenticator) authData(t *testing.T, attested bool) []byte {
	rpHash := sha256.Sum256([]byte(testHost))
	flags := byte(0x05) // user present and user verified
	if attested {
		flags |= 0x40
	}
	a.signCount++
	b := append([]byte{}, rpHash[:]...)
	b = append(b, flags)
	b = binary.BigEndian.AppendUint32(b, a.signCount)
	if attested {
		b = append(b, make([]byte, 16)...) // AAGUID
		b = binary.BigEndian.AppendUint16(b, uint16(len(a.credID)))
		b = append(b, a.credID...)
		b = append(b, a.coseKey(t)...)
	}
	return b
}

func clientData(typ, challenge string) []byte {
	b, _ := json.Marshal(map[string]any{"type": typ, "challenge": challenge, "origin": testOrigin})
	return b
}

// register returns the attestation response for the given challenge
func (a *authenticator) register(t *testing.T, challenge string) string {
	att := cborHead(5, 3)
	att = append(append(att, cborText("fmt")...), cborText("none")...)
	att = append(append(att, cborText("attStmt")...), cborHead(5, 0)...)
	att = append(append(att, cborText("authData")...), cborBytes(a.authData(t, true))...)
	b, _ := json.Marshal(map[string]any{
		"id":    b64(a.credID),
		"rawId": b64(a.credID),
		"type":  "public-key",
		"response": map[string]string{
			"clientDataJSON":    b64(clientData("webauthn.create", challenge)),
			"attestationObject": b64(att),
		},
	})
	return string(b)
}

// login returns the assertion response for the given challenge
func (a *authenticator) login(t *testing.T, challenge string) string {
	cd := clientData("webauthn.get", challenge)
	ad := a.authData(t, false)
	cdHash := sha256.Sum256(cd)
	digest := sha256.Sum256(append(append([]byte{}, ad...), cdHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{
		"id":    b64(a.credID),
		"rawId": b64(a.credID),
		"type":  "public-key",
		"response": map[string]string{
			"clientDataJSON":    b64(cd),
			"authenticatorData": b64(ad),
			"signature":         b64(sig),
		},
	})
	return string(b)
}

// backends opens the user database backends that the tests are run with
var backends = map[string]func(dir string) (pinterface.IUserState, error){
	"Bolt": func(dir string) (pinterface.IUserState, error) {
		perm, err := bolt.NewWithConf(filepath.Join(dir, "webauthn.db"))
		if err != nil {
			return nil, err
		}
		return perm.UserState(), nil
	},
	"SQLite": func(dir string) (pinterface.IUserState, error) {
		perm, err := sqlite.NewWithConf(filepath.Join(dir, "webauthn.sqlite"))
		if err != nil {
			return nil, err
		}
		return perm.UserState(), nil
	},
}

// forEachBackend runs f for every backend, with the users alice and bob
func forEachBackend(t *testing.T, f func(t *testing.T, us pinterface.IUserState)) {
	for name, open := range backends {
		t.Run(name, func(t *testing.T) {
			us, err := open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			us.AddUser("alice", "alicepassword", "alice@example.com")
			us.AddUser("bob", "bobpassword", "bob@example.com")
			f(t, us)
		})
	}
}

// loginCookies returns the cookies that a logged in user would send
func loginCookies(t *testing.T, us pinterface.IUserState, username string) []*http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	if err := us.Login(w, username); err != nil {
		t.Fatal(err)
	}
	return w.Result().Cookies()
}

// call runs one request that calls a WebAuthn Lua function, like
// `WebAuthnBeginLogin("alice")`, and returns the result and the response
func call(t *testing.T, us pinterface.IUserState, cookies []*http.Cookie, body, code string) (bool, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, testOrigin+"/", strings.NewReader(body))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	L := lua.NewState()
	defer L.Close()
	Load(w, req, L, us)
	if err := L.DoString("result = " + code); err != nil {
		t.Fatal(err)
	}
	return L.GetGlobal("result") == lua.LTrue, w
}

func challengeFrom(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var options struct {
		PublicKey struct {
			Challenge string `json:"challenge"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &options); err != nil || options.PublicKey.Challenge == "" {
		t.Fatalf("no challenge in %q: %v", w.Body.String(), err)
	}
	return options.PublicKey.Challenge
}

// registerPasskey registers a passkey for username, sending the given cookies
func registerPasskey(t *testing.T, us pinterface.IUserState, cookies []*http.Cookie, a *authenticator, username string) bool {
	t.Helper()
	ok, w := call(t, us, cookies, "", `WebAuthnBeginRegister("`+username+`")`)
	if !ok {
		return false
	}
	ok, _ = call(t, us, cookies, a.register(t, challengeFrom(t, w)), `WebAuthnFinishRegister("`+username+`")`)
	return ok
}

// loginWithPasskey logs in as username, and returns the session cookies, if any
func loginWithPasskey(t *testing.T, us pinterface.IUserState, a *authenticator, username string) (bool, []*http.Cookie) {
	t.Helper()
	ok, w := call(t, us, nil, "", `WebAuthnBeginLogin("`+username+`")`)
	if !ok {
		return false, nil
	}
	ok, w = call(t, us, nil, a.login(t, challengeFrom(t, w)), `WebAuthnFinishLogin("`+username+`")`)
	return ok, w.Result().Cookies()
}

func TestPasskeyRegisterAndLogin(t *testing.T) {
	forEachBackend(t, func(t *testing.T, us pinterface.IUserState) {
		a := newAuthenticator(t)
		if !registerPasskey(t, us, loginCookies(t, us, "alice"), a, "alice") {
			t.Fatal("alice could not register a passkey for her own account")
		}
		us.Logout("alice")
		ok, cookies := loginWithPasskey(t, us, a, "alice")
		if !ok || len(cookies) == 0 {
			t.Fatal("alice could not log in with her passkey")
		}
		if has, _ := us.Users().Has("alice", sessionField); has {
			t.Error("the WebAuthn session was not removed after logging in")
		}
	})
}

// Nobody may add their own passkey to someone else's account, and then use it
// to log in as that user.
func TestPasskeyAccountTakeover(t *testing.T) {
	forEachBackend(t, func(t *testing.T, us pinterface.IUserState) {
		attacker := newAuthenticator(t)

		for name, cookies := range map[string][]*http.Cookie{
			"not logged in":    nil,
			"logged in as bob": loginCookies(t, us, "bob"),
		} {
			if registerPasskey(t, us, cookies, attacker, "alice") {
				t.Errorf("%s: a passkey was registered for alice", name)
			}
		}
		if ok, _ := loginWithPasskey(t, us, attacker, "alice"); ok {
			t.Error("the attacker logged in as alice")
		}
		if us.IsLoggedIn("alice") {
			t.Error("alice is logged in")
		}
	})
}

// Passkeys are only for existing accounts, and unknown usernames must not
// leave anything behind in the user database.
func TestPasskeyUnknownUser(t *testing.T) {
	forEachBackend(t, func(t *testing.T, us pinterface.IUserState) {
		before, err := us.AllUsernames()
		if err != nil {
			t.Fatal(err)
		}
		a := newAuthenticator(t)
		if registerPasskey(t, us, nil, a, "nobody") {
			t.Error("a passkey was registered for a user that does not exist")
		}
		if ok, _ := loginWithPasskey(t, us, a, "nobody"); ok {
			t.Error("logged in as a user that does not exist")
		}
		after, err := us.AllUsernames()
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) || us.HasUser("nobody") {
			t.Errorf("users before: %v, after: %v", before, after)
		}
	})
}

// Credentials stored by older versions of Algernon
func TestPasskeyLegacyCredentials(t *testing.T) {
	forEachBackend(t, func(t *testing.T, us pinterface.IUserState) {
		a := newAuthenticator(t)
		if !registerPasskey(t, us, loginCookies(t, us, "alice"), a, "alice") {
			t.Fatal("alice could not register a passkey for her own account")
		}
		us.Logout("alice")

		raw, err := us.Users().Get("alice", credentialsField)
		if err != nil {
			t.Fatal(err)
		}
		if err := us.Users().DelKey("alice", credentialsField); err != nil {
			t.Fatal(err)
		}
		legacy, err := us.Creator().NewKeyValue(legacyCredStore)
		if err != nil {
			t.Skip("this backend could not hold the legacy credentials either:", err)
		}
		if err := legacy.Set("alice", raw); err != nil {
			t.Fatal(err)
		}

		if ok, _ := loginWithPasskey(t, us, a, "alice"); !ok {
			t.Fatal("alice could not log in with a passkey from the legacy store")
		}
		if moved, _ := us.Users().Get("alice", credentialsField); moved != raw {
			t.Error("the credentials were not moved to the user")
		}
		if left, _ := legacy.Get("alice"); left != "" {
			t.Error("the credentials were not removed from the legacy store")
		}
	})
}
