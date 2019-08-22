package permissions

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/xyproto/cookie"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simpleredis"
)

const (
	defaultRedisServer = ":6379"
)

var (
	minConfirmationCodeLength = 20 // minimum length of the confirmation code

	// ErrNotFound is used as an error if not finding what is being searched for
	ErrNotFound = errors.New("not found")
)

// UserState is a struct for dealing with the user state, users and passwords.
// Can also be used for retrieving the underlying Redis connection pool.
// The default password hashing algorithm is "bcrypt+", which is the same as
// "bcrypt", but with backwards compatibility for checking sha256 hashes.
type UserState struct {
	// see: http://redis.io/topics/data-types
	users             *simpleredis.HashMap        // Hash map of users, with several different fields per user ("loggedin", "confirmed", "email" etc)
	usernames         *simpleredis.Set            // A list of all usernames, for easy enumeration
	unconfirmed       *simpleredis.Set            // A list of unconfirmed usernames, for easy enumeration
	pool              *simpleredis.ConnectionPool // A connection pool for Redis
	dbindex           int                         // Redis database index
	cookieSecret      string                      // Secret for storing secure cookies
	cookieTime        int64                       // How long a cookie should last, in seconds
	passwordAlgorithm string                      // Password hashing algorithm ("sha256", "bcrypt" or "bcrypt+").
}

// NewUserStateSimple will create a new *UserState that can be used for
// managing users. The random number generator will be seeded after generating
// the cookie secret. A connection pool for the local Redis server (dbindex 0)
// will be created. Calls log.Fatal if things go wrong.
func NewUserStateSimple() *UserState {
	// db index 0, initialize random generator after generating the cookie secret
	return NewUserState(0, true, defaultRedisServer)
}

// NewUserStateSimple2 will create a new *UserState that can be used for
// managing users. The random number generator will be seeded after generating
// the cookie secret. A connection pool for the local Redis server (dbindex 0)
// will be created. Returns an error if things go wrong.
func NewUserStateSimple2() (*UserState, error) {
	// db index 0, initialize random generator after generating the cookie secret
	return NewUserState2(0, true, defaultRedisServer)
}

// NewUserStateWithPassword is the same as NewUserStateSimple, but also takes
// a Redis hostname and a Redis password. Use NewUserState for control over the
// database index and port number. Calls log.Fatal if things go wrong.
func NewUserStateWithPassword(hostname, password string) *UserState {
	// db index 0, initialize random generator after generating the cookie secret, password
	var connectTo string
	switch {
	case (password == "") && (strings.Count(hostname, ":") == 0):
		connectTo = hostname + ":6379"
	case strings.Count(hostname, ":") > 0:
		connectTo = password + "@" + hostname
	default:
		connectTo = password + "@" + hostname + ":6379"
	}
	// Create a new UserState with database index 0, "true" for seeding the
	// random number generator and host string
	return NewUserState(0, true, connectTo)
}

// NewUserStateWithPassword2 is the same as NewUserStateSimple2, but takes
// a hostname and a password. Use NewUserState2 for control over the database
// index and port number. Returns an error if things go wrong.
func NewUserStateWithPassword2(hostname, password string) (*UserState, error) {
	// db index 0, initialize random generator after generating the cookie secret, password
	var connectTo string
	if (password == "") && (strings.Count(hostname, ":") == 0) {
		connectTo = hostname + ":6379"
	} else if strings.Count(hostname, ":") > 0 {
		connectTo = password + "@" + hostname
	} else {
		connectTo = password + "@" + hostname + ":6379"
	}
	// Create a new UserState with database index 0, "true" for seeding the
	// random number generator and host string
	return NewUserState2(0, true, connectTo)
}

// NewUserState will create a new *UserState that can be used for managing
// users. dbindex is the Redis database index (0 is a good default value).
// If randomseed is true, the random number generator will be seeded after
// generating the cookie secret (true is a good default value).
// redisHostPort is host:port for the desired Redis server (can be blank for
// localhost). Also creates a new ConnectionPool. Calls log.Fatal if things
// go wrong.
func NewUserState(dbindex int, randomseed bool, redisHostPort string) *UserState {
	var pool *simpleredis.ConnectionPool

	// Connect to the default redis server if redisHostPort is empty
	if redisHostPort == "" {
		redisHostPort = defaultRedisServer
	}

	// Test connection (the client can do this test before creating a new userstate)
	if err := simpleredis.TestConnectionHost(redisHostPort); err != nil {
		errorMessage := err.Error()
		if errorMessage == "dial tcp :6379: getsockopt: connection refused" {
			log.Fatalln("Fatal: Unable to connect to Redis server on port 6379.")
		}
		log.Fatalln(errorMessage)
	}

	// Acquire connection pool
	pool = simpleredis.NewConnectionPoolHost(redisHostPort)

	state := new(UserState)

	state.users = simpleredis.NewHashMap(pool, "users")
	state.users.SelectDatabase(dbindex)

	state.usernames = simpleredis.NewSet(pool, "usernames")
	state.usernames.SelectDatabase(dbindex)

	state.unconfirmed = simpleredis.NewSet(pool, "unconfirmed")
	state.unconfirmed.SelectDatabase(dbindex)

	state.pool = pool

	state.dbindex = dbindex

	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	state.cookieSecret = cookie.RandomCookieFriendlyString(30)

	// Seed the random number generator
	if randomseed {
		rand.Seed(time.Now().UnixNano())
	}

	// Cookies lasts for 24 hours by default. Specified in seconds.
	state.cookieTime = cookie.DefaultCookieTime

	// Default password hashing algorithm is "bcrypt+", which is the same as
	// "bcrypt", but with backwards compatibility for checking sha256 hashes.
	state.passwordAlgorithm = "bcrypt+" // "bcrypt+", "bcrypt" or "sha256"

	if pool.Ping() != nil {
		defer pool.Close()
		log.Fatalf("Error, wrong hostname, port or password. (%s does not reply to PING)\n", redisHostPort)
	}

	return state
}

// NewUserState2 will create a new *UserState that can be used for managing users.
// dbindex is the Redis database index (0 is a good default value).
// If randomseed is true, the random number generator will be seeded after generating the cookie secret (true is a good default value).
// redisHostPort is host:port for the desired Redis server (can be blank for localhost)
// Also creates a new ConnectionPool.
// Returns an error if things go wrong.
func NewUserState2(dbindex int, randomseed bool, redisHostPort string) (*UserState, error) {
	var pool *simpleredis.ConnectionPool

	// Connect to the default redis server if redisHostPort is empty
	if redisHostPort == "" {
		redisHostPort = defaultRedisServer
	}

	// Test connection (the client can do this test before creating a new userstate)
	if err := simpleredis.TestConnectionHost(redisHostPort); err != nil {
		errorMessage := err.Error()
		if errorMessage == "dial tcp :6379: getsockopt: connection refused" {
			return nil, errors.New("unable to connect to Redis server on port 6379")
		}
		return nil, err
	}

	// Acquire connection pool
	pool = simpleredis.NewConnectionPoolHost(redisHostPort)

	state := new(UserState)

	state.users = simpleredis.NewHashMap(pool, "users")
	state.users.SelectDatabase(dbindex)

	state.usernames = simpleredis.NewSet(pool, "usernames")
	state.usernames.SelectDatabase(dbindex)

	state.unconfirmed = simpleredis.NewSet(pool, "unconfirmed")
	state.unconfirmed.SelectDatabase(dbindex)

	state.pool = pool

	state.dbindex = dbindex

	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	state.cookieSecret = cookie.RandomCookieFriendlyString(30)

	// Seed the random number generator
	if randomseed {
		rand.Seed(time.Now().UnixNano())
	}

	// Cookies lasts for 24 hours by default. Specified in seconds.
	state.cookieTime = cookie.DefaultCookieTime

	// Default password hashing algorithm is "bcrypt+", which is the same as
	// "bcrypt", but with backwards compatibility for checking sha256 hashes.
	state.passwordAlgorithm = "bcrypt+" // "bcrypt+", "bcrypt" or "sha256"

	if pool.Ping() != nil {
		defer pool.Close()
		return nil, fmt.Errorf("wrong hostname, port or password. (%s does not reply to PING)", redisHostPort)
	}

	return state, nil
}

// Host gets the Host (for qualifying for the IUserState interface)
func (state *UserState) Host() pinterface.IHost {
	return state.pool
}

// DatabaseIndex gets the Redis database index.
func (state *UserState) DatabaseIndex() int {
	return state.dbindex
}

// Pool gets the Redis connection pool.
func (state *UserState) Pool() *simpleredis.ConnectionPool {
	return state.pool
}

// Close the Redis connection pool.
func (state *UserState) Close() {
	state.pool.Close()
}

// UserRights checks if the current user is logged in and has user rights.
func (state *UserState) UserRights(req *http.Request) bool {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return false
	}
	return state.IsLoggedIn(username)
}

// HasUser checks if the given username exists.
func (state *UserState) HasUser(username string) bool {
	val, err := state.usernames.Has(username)
	if err != nil {
		// This happened at concurrent connections before introducing the connection pool
		panic("ERROR: Lost connection to Redis?")
	}
	return val
}

// HasUser2 checks if the given username exists.
func (state *UserState) HasUser2(username string) (bool, error) {
	val, err := state.usernames.Has(username)
	if err != nil {
		// This happened at concurrent connections before introducing the connection pool
		return false, errors.New("Lost connection to Redis?")
	}
	return val, nil
}

// HasEmail finds the user that has a given e-mail address.
// Returns the username and nil if found or a blank string and ErrNotFound if not.
func (state *UserState) HasEmail(email string) (string, error) {
	if email == "" {
		return "", ErrNotFound
	}
	usernames, err := state.AllUsernames()
	if err != nil {
		return "", err
	}
	for _, username := range usernames {
		userEmail, err := state.Email(username)
		if err != nil {
			return "", err
		}
		if userEmail == email {
			return username, nil
		}
	}
	return "", ErrNotFound
}

// BooleanField returns the boolean value for a given username and field name.
// If the user or field is missing, false will be returned.
// Useful for states where it makes sense that the returned value is not true
// unless everything is in order.
func (state *UserState) BooleanField(username, fieldname string) bool {
	hasUser := state.HasUser(username)
	if !hasUser {
		return false
	}
	value, err := state.users.Get(username, fieldname)
	if err != nil {
		return false
	}
	return value == "true"
}

// SetBooleanField can store a boolean value for the given username and custom fieldname.
func (state *UserState) SetBooleanField(username, fieldname string, val bool) {
	strval := "false"
	if val {
		strval = "true"
	}
	state.users.Set(username, fieldname, strval)
}

// IsConfirmed checks if the given username is confirmed.
func (state *UserState) IsConfirmed(username string) bool {
	return state.BooleanField(username, "confirmed")
}

// IsLoggedIn checks if the given username is logged in.
func (state *UserState) IsLoggedIn(username string) bool {
	if !state.HasUser(username) {
		return false
	}
	status, err := state.users.Get(username, "loggedin")
	if err != nil {
		// Returns "no" if the status can not be retrieved
		return false
	}
	return status == "true"
}

// AdminRights checks if the current user is logged in and has administrator rights.
func (state *UserState) AdminRights(req *http.Request) bool {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return false
	}
	return state.IsLoggedIn(username) && state.IsAdmin(username)
}

// IsAdmin checks if the given username is an administrator.
func (state *UserState) IsAdmin(username string) bool {
	if !state.HasUser(username) {
		return false
	}
	status, err := state.users.Get(username, "admin")
	if err != nil {
		return false
	}
	return status == "true"
}

// UsernameCookie retrieves the username that is stored in a cookie in the browser, if available.
func (state *UserState) UsernameCookie(req *http.Request) (string, error) {
	username, ok := cookie.SecureCookie(req, "user", state.cookieSecret)
	if ok && (username != "") {
		return username, nil
	}
	return "", errors.New("Could not retrieve the username from browser cookie")
}

// Store the given username in a cookie in the browser, if possible.
// The user must exist.
// There are two cookie flags (ref RFC6265: https://tools.ietf.org/html/rfc6265#section-5.2.5):
// - secure is for only allowing cookies to be set over HTTPS
// - httponly is for only allowing cookies for the same server
func (state *UserState) setUsernameCookieWithFlags(w http.ResponseWriter, username string, secure, httponly bool) error {
	if username == "" {
		return errors.New("Can't set cookie for empty username")
	}
	if !state.HasUser(username) {
		return errors.New("Can't store cookie for non-existing user")
	}
	// Create a cookie that lasts for a while ("timeout" seconds),
	// this is the equivalent of a session for a given username.
	cookie.SetSecureCookiePathWithFlags(w, "user", username, state.cookieTime, "/", state.cookieSecret, false, true)
	return nil
}

/*SetUsernameCookie tries to store the given username in a cookie in the browser.
 *
 * The user must exist. Returns an error if the username is empty or does not exist.
 * Returns nil if the cookie has been attempted to be set.
 * To check if the cookie has actually been set, one must try to read it.
 */
func (state *UserState) SetUsernameCookie(w http.ResponseWriter, username string) error {
	// These cookie flags are set (ref RFC6265)
	// "secure" is set to false (only allow cookies to be set over HTTPS)
	// "httponly" is set to true (only allow cookies being set/read from the same server)
	return state.setUsernameCookieWithFlags(w, username, false, true)
}

/*SetUsernameCookieOnlyHTTPS tries to store the given username in a cookie in the browser.
 * This function will not set the cookie if over plain HTTP.
 *
 * The user must exist. Returns an error if the username is empty or does not exist.
 * Returns nil if the cookie has been attempted to be set.
 * To check if the cookie has actually been set, one must try to read it.
 */
func (state *UserState) SetUsernameCookieOnlyHTTPS(w http.ResponseWriter, username string) error {
	// These cookie flags are set (ref RFC6265)
	// "secure" is set to true (only allow cookies to be set over HTTPS)
	// "httponly" is set to true (only allow cookies being set/read from the same server)
	return state.setUsernameCookieWithFlags(w, username, true, true)
}

// AllUsernames retrieves a list of all usernames.
func (state *UserState) AllUsernames() ([]string, error) {
	return state.usernames.GetAll()
}

// Email returns the email address for the given username.
func (state *UserState) Email(username string) (string, error) {
	return state.users.Get(username, "email")
}

// PasswordHash returns the password hash for the given username.
func (state *UserState) PasswordHash(username string) (string, error) {
	return state.users.Get(username, "password")
}

// AllUnconfirmedUsernames returns a list of all registered users that are not yet confirmed.
func (state *UserState) AllUnconfirmedUsernames() ([]string, error) {
	return state.unconfirmed.GetAll()
}

// ConfirmationCode gets the confirmation code for a specific user.
func (state *UserState) ConfirmationCode(username string) (string, error) {
	return state.users.Get(username, "confirmationCode")
}

// Users gets the users HashMap.
func (state *UserState) Users() pinterface.IHashMap {
	return state.users
}

// AddUnconfirmed adds a user that is registered but not confirmed.
func (state *UserState) AddUnconfirmed(username, confirmationCode string) {
	state.unconfirmed.Add(username)
	state.users.Set(username, "confirmationCode", confirmationCode)
}

// RemoveUnconfirmed removes a user that is registered but not confirmed.
func (state *UserState) RemoveUnconfirmed(username string) {
	state.unconfirmed.Del(username)
	state.users.DelKey(username, "confirmationCode")
}

// MarkConfirmed can mark a user as confirmed.
func (state *UserState) MarkConfirmed(username string) {
	state.users.Set(username, "confirmed", "true")
}

// RemoveUser removes user and login status.
func (state *UserState) RemoveUser(username string) {
	state.usernames.Del(username)
	// Remove additional data as well
	// TODO: Ideally, remove all keys belonging to the user.
	state.users.DelKey(username, "loggedin")
}

// SetAdminStatus can make a user an administrator.
func (state *UserState) SetAdminStatus(username string) {
	state.users.Set(username, "admin", "true")
}

// RemoveAdminStatus can remove administrator status from a user.
func (state *UserState) RemoveAdminStatus(username string) {
	state.users.Set(username, "admin", "false")
}

// SetToken sets a token for a user, for a given expiry time.
func (state *UserState) SetToken(username, token string, expire time.Duration) {
	state.users.SetExpire(username, "token", token, expire)
}

// GetToken retrieves the token for a user.
func (state *UserState) GetToken(username string) (string, error) {
	return state.users.Get(username, "token")
}

// RemoveToken takes a username and removes the associated token.
func (state *UserState) RemoveToken(username string) {
	state.users.DelKey(username, "token")
}

// SetPassword sets the password for a user. The given password string will be hashed.
// No validation or check of the given password is performed.
func (state *UserState) SetPassword(username, password string) {
	state.users.Set(username, "password", state.HashPassword(username, password))
}

// Creates a user from the username and password hash, does not check for rights.
func (state *UserState) addUserUnchecked(username, passwordHash, email string) {
	// Add the user
	state.usernames.Add(username)

	// Add password and email
	state.users.Set(username, "password", passwordHash)
	state.users.Set(username, "email", email)

	// Additional fields
	additionalfields := []string{"loggedin", "confirmed", "admin"}
	for _, fieldname := range additionalfields {
		state.users.Set(username, fieldname, "false")
	}
}

// AddUser creates a user and hashes the password, does not check for rights.
// The given data must be valid.
func (state *UserState) AddUser(username, password, email string) {
	passwordHash := state.HashPassword(username, password)
	state.addUserUnchecked(username, passwordHash, email)
}

// SetLoggedIn will mark the user as logged in. Use the Login function instead, unless cookies are not involved.
func (state *UserState) SetLoggedIn(username string) {
	state.users.Set(username, "loggedin", "true")
}

// SetLoggedOut will mark the user as logged out.
func (state *UserState) SetLoggedOut(username string) {
	state.users.Set(username, "loggedin", "false")
}

// Login is a convenience function for logging a user in and storing the username in a cookie.
// Returns an error if the cookie could not be set.
func (state *UserState) Login(w http.ResponseWriter, username string) error {
	state.SetLoggedIn(username)
	return state.SetUsernameCookie(w, username)
}

// ClearCookie will try to clear the user cookie by setting it to expired.
// Some browsers *may* be configured to keep cookies even after this, but that is highly unusual.
func (state *UserState) ClearCookie(w http.ResponseWriter) {
	cookie.ClearCookie(w, "user", "/")
}

// Logout is a convenience function for logging a user out. This is the same as SetLoggedOut.
func (state *UserState) Logout(username string) {
	state.SetLoggedOut(username)
}

// Username is a convenience function that will return a username (from the browser cookie) or an empty string.
func (state *UserState) Username(req *http.Request) string {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return ""
	}
	return username
}

// CookieTimeout gets how long a login cookie should last, in seconds.
func (state *UserState) CookieTimeout(username string) int64 {
	return state.cookieTime
}

// SetCookieTimeout will set how long a login cookie should last, in seconds.
func (state *UserState) SetCookieTimeout(cookieTime int64) {
	state.cookieTime = cookieTime
}

// CookieSecret returns the current cookie secret.
func (state *UserState) CookieSecret() string {
	return state.cookieSecret
}

// SetCookieSecret will set the secret that is used when generating secure cookies.
func (state *UserState) SetCookieSecret(cookieSecret string) {
	state.cookieSecret = cookieSecret
}

// PasswordAlgo gets the current password hashing algorithm.
func (state *UserState) PasswordAlgo() string {
	return state.passwordAlgorithm
}

// SetPasswordAlgo can set the password hashing algorithm that should be used.
// The default is "bcrypt+".
// Possible values are:
//    bcrypt  -> Store and check passwords with the bcrypt hash.
//    sha256  -> Store and check passwords with the sha256 hash.
//    bcrypt+ -> Store passwords with bcrypt, but check with both
//               bcrypt and sha256, for backwards compatibility
//               with old passwords that has been stored as sha256.
func (state *UserState) SetPasswordAlgo(algorithm string) error {
	switch algorithm {
	case "sha256", "bcrypt", "bcrypt+":
		state.passwordAlgorithm = algorithm
	default:
		return errors.New("Permissions: " + algorithm + " is an unsupported encryption algorithm")
	}
	return nil
}

// HashPassword will hash the password (takes a username as well, it can be used for salting when using sha256).
func (state *UserState) HashPassword(username, password string) string {
	switch state.passwordAlgorithm {
	case "sha256":
		return string(hashSha256(state.cookieSecret, username, password))
	case "bcrypt", "bcrypt+":
		return string(hashBcrypt(password))
	}
	// Only valid password algorithms should be allowed to set
	return ""
}

// storedHash returns the stored hash, or an empty byte slice.
func (state *UserState) storedHash(username string) []byte {
	hashString, err := state.PasswordHash(username)
	if err != nil {
		return []byte{}
	}
	return []byte(hashString)
}

// CorrectPassword checks if a password is correct. username is needed because it is part of the hash.
func (state *UserState) CorrectPassword(username, password string) bool {

	if !state.HasUser(username) {
		return false
	}

	// Retrieve the stored password hash
	hash := state.storedHash(username)
	if len(hash) == 0 {
		return false
	}

	// Check the password with the right password algorithm
	switch state.passwordAlgorithm {
	case "sha256":
		return correctSha256(hash, state.cookieSecret, username, password)
	case "bcrypt":
		return correctBcrypt(hash, password)
	case "bcrypt+": // for backwards compatibility with sha256
		if isSha256(hash) && correctSha256(hash, state.cookieSecret, username, password) {
			return true
		}
		return correctBcrypt(hash, password)
	}
	return false
}

// AlreadyHasConfirmationCode runs through all confirmation codes of all unconfirmed
// users and checks if this confirmationCode is already in use.
func (state *UserState) AlreadyHasConfirmationCode(confirmationCode string) bool {
	unconfirmedUsernames, err := state.AllUnconfirmedUsernames()
	if err != nil {
		return false
	}
	for _, aUsername := range unconfirmedUsernames {
		aConfirmationCode, err := state.ConfirmationCode(aUsername)
		if err != nil {
			// If the confirmation code can not be found, that's okay too
			return false
		}
		if confirmationCode == aConfirmationCode {
			// Found it
			return true
		}
	}
	return false
}

// FindUserByConfirmationCode can find the corresponding username in the list
// of unconfirmed users, given a unique confirmation code.
func (state *UserState) FindUserByConfirmationCode(confirmationCode string) (string, error) {
	unconfirmedUsernames, err := state.AllUnconfirmedUsernames()
	if err != nil {
		return "", errors.New("all existing users are already confirmed")
	}

	// Find the username by looking up the confirmationCode on unconfirmed users
	username := ""
	for _, aUsername := range unconfirmedUsernames {
		aConfirmationCode, err := state.ConfirmationCode(aUsername)
		if err != nil {
			// If the confirmation code can not be found, just skip this one
			continue
		}
		if confirmationCode == aConfirmationCode {
			// Found the right user
			username = aUsername
			break
		}
	}

	// Check that the user is there
	if username == "" {
		return username, errors.New("the confirmation code is no longer valid")
	}
	hasUser := state.HasUser(username)
	if !hasUser {
		return username, errors.New("the user that is to be confirmed no longer exists")
	}

	return username, nil
}

// Confirm removes the username from the list of unconfirmed users and mark the user as confirmed.
func (state *UserState) Confirm(username string) {
	// Remove from the list of unconfirmed usernames
	state.RemoveUnconfirmed(username)

	// Mark user as confirmed
	state.MarkConfirmed(username)
}

// ConfirmUserByConfirmationCode takes a confirmation code and mark the corresponding unconfirmed user as confirmed.
func (state *UserState) ConfirmUserByConfirmationCode(confirmationCode string) error {
	username, err := state.FindUserByConfirmationCode(confirmationCode)
	if err != nil {
		return err
	}
	state.Confirm(username)
	return nil
}

// SetMinimumConfirmationCodeLength will set the minimum length of the user
// confirmation code. The default is 20.
func (state *UserState) SetMinimumConfirmationCodeLength(length int) {
	minConfirmationCodeLength = length
}

// GenerateUniqueConfirmationCode will generate a unique confirmation code that
// can be used for confirming users after users have registered.
func (state *UserState) GenerateUniqueConfirmationCode() (string, error) {
	const maxConfirmationCodeLength = 100 // when are the generated confirmation codes unreasonably long
	length := minConfirmationCodeLength
	confirmationCode := cookie.RandomHumanFriendlyString(length)
	for state.AlreadyHasConfirmationCode(confirmationCode) {
		// Increase the length of the confirmationCode random string every time there is a collision
		length++
		confirmationCode = cookie.RandomHumanFriendlyString(length)
		if length > maxConfirmationCodeLength {
			// This should never happen
			return confirmationCode, errors.New("too many generated confirmation codes are not unique")
		}
	}
	return confirmationCode, nil
}

// ValidUsernamePassword checks that the given username and password are different.
// Also check if the chosen username only contains letters, numbers and/or underscore.
// Use the "CorrectPassword" function for checking if the password is correct.
func ValidUsernamePassword(username, password string) error {
	const allAllowedLetters = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
NEXT:
	for _, letter := range username {
		for _, allowedLetter := range allAllowedLetters {
			if letter == allowedLetter {
				continue NEXT // check the next letter in the username
			}
		}
		return errors.New("only letters, numbers and underscore are allowed in usernames")
	}
	if username == password {
		return errors.New("username and password must be different, try another password")
	}
	return nil
}

// Creator returns a struct for creating data structures with
func (state *UserState) Creator() pinterface.ICreator {
	return simpleredis.NewCreator(state.pool, state.dbindex)
}

// Properties returns a list of user properties.
// Returns an empty list if the user has no properties, or if there are errors.
func (state *UserState) Properties(username string) []string {
	props, err := state.users.Keys(username)
	if err != nil {
		return []string{}
	}
	return props
}
