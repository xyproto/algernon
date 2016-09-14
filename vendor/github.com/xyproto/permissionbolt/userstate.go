package permissionbolt

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/xyproto/cookie"     // Functions related to cookies
	"github.com/xyproto/pinterface" // Database interfaces
	"github.com/xyproto/simplebolt" // Bolt database wrapper
)

const (
	defaultFilename = "bolt.db"
)

var (
	minConfirmationCodeLength = 20 // minimum length of the confirmation code
)

// The UserState struct holds the pointer to the underlying database and a few other settings
type UserState struct {
	users             *simplebolt.HashMap  // Hash map of users, with several different fields per user ("loggedin", "confirmed", "email" etc)
	usernames         *simplebolt.Set      // A list of all usernames, for easy enumeration
	unconfirmed       *simplebolt.Set      // A list of unconfirmed usernames, for easy enumeration
	db                *simplebolt.Database // A Bolt database
	cookieSecret      string               // Secret for storing secure cookies
	cookieTime        int64                // How long a cookie should last, in seconds
	passwordAlgorithm string               // The hashing algorithm to utilize default: "bcrypt+" allowed: ("sha256", "bcrypt", "bcrypt+")
}

// NewUserStateSimple creates a new UserState struct that can be used for managing users.
// The random number generator will be seeded after generating the cookie secret.
func NewUserStateSimple() (*UserState, error) {
	// connection string | initialize random generator after generating the cookie secret
	return NewUserState(defaultFilename, true)
}

// NewUserState creates a new UserState struct that can be used for managing users.
// connectionString may be on the form "username:password@host:port/database".
// If randomseed is true, the random number generator will be seeded after generating the cookie secret
// (true is a good default value).
func NewUserState(filename string, randomseed bool) (*UserState, error) {
	var err error

	db, err := simplebolt.New(filename)
	if err != nil {
		return nil, err
	}

	state := new(UserState)

	state.users, err = simplebolt.NewHashMap(db, "users")
	if err != nil {
		return nil, err
	}
	state.usernames, err = simplebolt.NewSet(db, "usernames")
	if err != nil {
		return nil, err
	}
	state.unconfirmed, err = simplebolt.NewSet(db, "unconfirmed")
	if err != nil {
		return nil, err
	}

	state.db = db

	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	state.cookieSecret = cookie.RandomCookieFriendlyString(30)

	// Seed the random number generator
	if randomseed {
		rand.Seed(time.Now().UnixNano())
	}

	// Cookies lasts for 24 hours by default. Specified in seconds.
	state.cookieTime = 3600 * 24

	// Default password hashing algorithm is "bcrypt+", which is the same as
	// "bcrypt", but with backwards compatibility for checking sha256 hashes.
	state.passwordAlgorithm = "bcrypt+" // "bcrypt+", "bcrypt" or "sha256"

	return state, nil
}

// Database retrieves the underlying database
func (state *UserState) Database() *simplebolt.Database {
	return state.db
}

// Host retrieves the underlying database. It helps fulfill the IHost interface.
func (state *UserState) Host() pinterface.IHost {
	return state.db
}

// Close the connection to the database host
func (state *UserState) Close() {
	state.db.Close()
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
		panic("ERROR: Lost connection to database?")
	}
	return val
}

// BooleanField returns a boolean value for the given username and fieldname.
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

// SetBooleanField stores a boolean value given a username and a custom fieldname.
func (state *UserState) SetBooleanField(username, fieldname string, val bool) {
	strval := "false"
	if val {
		strval = "true"
	}
	state.users.Set(username, fieldname, strval)
}

// IsConfirmed checks if a user is confirmed (can be used for "e-mail confirmation").
func (state *UserState) IsConfirmed(username string) bool {
	return state.BooleanField(username, "confirmed")
}

// IsLoggedIn checks if a user is logged in.
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

// IsAdmin checks if a user is an administrator.
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

// SetUsernameCookie stores the given username in a cookie in the browser, if possible.
// Will return an error if the username is empty or the user does not exist.
func (state *UserState) SetUsernameCookie(w http.ResponseWriter, username string) error {
	if username == "" {
		return errors.New("Can't set cookie for empty username")
	}
	if !state.HasUser(username) {
		return errors.New("Can't store cookie for non-existsing user")
	}
	// Create a cookie that lasts for a while ("timeout" seconds),
	// this is the equivivalent of a session for a given username.
	cookie.SetSecureCookiePath(w, "user", username, state.cookieTime, "/", state.cookieSecret)
	return nil
}

// AllUsernames returns a list of all usernames.
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

// ConfirmationCode returns the stored confirmation code for a specific user.
func (state *UserState) ConfirmationCode(username string) (string, error) {
	return state.users.Get(username, "confirmationCode")
}

// AddUnconfirmed adds a user to a list of users that are registered, but not confirmed.
func (state *UserState) AddUnconfirmed(username, confirmationCode string) {
	state.unconfirmed.Add(username)
	state.users.Set(username, "confirmationCode", confirmationCode)
}

// RemoveUnconfirmed removes a user from a list of users that are registered, but not confirmed.
func (state *UserState) RemoveUnconfirmed(username string) {
	state.unconfirmed.Del(username)
	state.users.DelKey(username, "confirmationCode")
}

// MarkConfirmed marks a user as being confirmed.
func (state *UserState) MarkConfirmed(username string) {
	state.users.Set(username, "confirmed", "true")
}

// RemoveUser removes a user and the login status for this user.
func (state *UserState) RemoveUser(username string) {
	state.usernames.Del(username)
	// Remove additional data as well
	//state.users.DelKey(username, "loggedin")
	state.users.Del(username)
}

// SetAdminStatus marks a user as an administrator.
func (state *UserState) SetAdminStatus(username string) {
	state.users.Set(username, "admin", "true")
}

// RemoveAdminStatus removes the administrator status from a user.
func (state *UserState) RemoveAdminStatus(username string) {
	state.users.Set(username, "admin", "false")
}

// addUserUnchecked creates a user from the username and password hash, does not check for rights.
func (state *UserState) addUserUnchecked(username, passwordHash, email string) {
	// Add the user
	state.usernames.Add(username)

	// Add password and email
	state.users.Set(username, "password", passwordHash)
	state.users.Set(username, "email", email)

	// Addditional fields
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

// SetLoggedIn marks a user as logged in.
// Use the Login function instead, unless cookies are not involved.
func (state *UserState) SetLoggedIn(username string) {
	state.users.Set(username, "loggedin", "true")
}

// SetLoggedOut marks a user as logged out.
func (state *UserState) SetLoggedOut(username string) {
	state.users.Set(username, "loggedin", "false")
}

// Login is a convenience function for logging a user in and storing the
// username in a cookie. Returns an error if the cookie could not be set.
func (state *UserState) Login(w http.ResponseWriter, username string) error {
	state.SetLoggedIn(username)
	return state.SetUsernameCookie(w, username)
}

// ClearCookie tries to clear the user cookie by setting it to be expired.
// Some browsers *may* be configured to keep cookies even after this.
func (state *UserState) ClearCookie(w http.ResponseWriter) {
	cookie.ClearCookie(w, "user", "/")
}

// Logout is a convenience function for logging out a user.
func (state *UserState) Logout(username string) {
	state.SetLoggedOut(username)
}

// Username is a convenience function for returning the current username
// (from the browser cookie), or an empty string.
func (state *UserState) Username(req *http.Request) string {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return ""
	}
	return username
}

// CookieTimeout returns the current login cookie timeout, in seconds.
func (state *UserState) CookieTimeout(username string) int64 {
	return state.cookieTime
}

// SetCookieTimeout sets how long a login cookie should last, in seconds.
func (state *UserState) SetCookieTimeout(cookieTime int64) {
	state.cookieTime = cookieTime
}

// PasswordAlgo returns the current password hashing algorithm.
func (state *UserState) PasswordAlgo() string {
	return state.passwordAlgorithm
}

/*SetPasswordAlgo determines which password hashing algorithm should be used.
 *
 * The default value is "bcrypt+".
 *
 * Possible values are:
 *    bcrypt  -> Store and check passwords with the bcrypt hash.
 *    sha256  -> Store and check passwords with the sha256 hash.
 *    bcrypt+ -> Store passwords with bcrypt, but check with both
 *               bcrypt and sha256, for backwards compatibility
 *               with old passwords that has been stored as sha256.
 */
func (state *UserState) SetPasswordAlgo(algorithm string) error {
	switch algorithm {
	case "sha256", "bcrypt", "bcrypt+":
		state.passwordAlgorithm = algorithm
	default:
		return errors.New("Permissions: " + algorithm + " is an unsupported encryption algorithm")
	}
	return nil
}

// HashPassword takes a password and creates a password hash.
// It also takes a username, since some algorithms may use it for salt.
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

// SetPassword sets/changes the password for a user.
// Does not take a password hash, will hash the password string.
func (state *UserState) SetPassword(username, password string) {
	state.users.Set(username, "password", state.HashPassword(username, password))
}

// Return the stored hash, or an empty byte slice.
func (state *UserState) storedHash(username string) []byte {
	hashString, err := state.PasswordHash(username)
	if err != nil {
		return []byte{}
	}
	return []byte(hashString)
}

// CorrectPassword checks if a password is correct. "username" is needed because
// it may be part of the hash for some password hashing algorithms.
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

// AlreadyHasConfirmationCode goes through all the confirmationCodes of all
// the unconfirmed users and checks if this confirmationCode already is in use.
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

// FindUserByConfirmationCode tries to find the corresponding username,
// given a unique confirmation code.
func (state *UserState) FindUserByConfirmationCode(confirmationcode string) (string, error) {
	unconfirmedUsernames, err := state.AllUnconfirmedUsernames()
	if err != nil {
		return "", errors.New("All existing users are already confirmed.")
	}

	// Find the username by looking up the confirmationcode on unconfirmed users
	username := ""
	for _, aUsername := range unconfirmedUsernames {
		aConfirmationCode, err := state.ConfirmationCode(aUsername)
		if err != nil {
			// If the confirmation code can not be found, just skip this one
			continue
		}
		if confirmationcode == aConfirmationCode {
			// Found the right user
			username = aUsername
			break
		}
	}

	// Check that the user is there
	if username == "" {
		return username, errors.New("The confirmation code is no longer valid.")
	}
	hasUser := state.HasUser(username)
	if !hasUser {
		return username, errors.New("The user that is to be confirmed no longer exists.")
	}

	return username, nil
}

// Confirm marks a user as confirmed, and removes the username from the list
// of unconfirmed users.
func (state *UserState) Confirm(username string) {
	// Remove from the list of unconfirmed usernames
	state.RemoveUnconfirmed(username)

	// Mark user as confirmed
	state.MarkConfirmed(username)
}

// ConfirmUserByConfirmationCode takes a unique confirmation code and marks
// the corresponding unconfirmed user as confirmed.
func (state *UserState) ConfirmUserByConfirmationCode(confirmationcode string) error {
	username, err := state.FindUserByConfirmationCode(confirmationcode)
	if err != nil {
		return err
	}
	state.Confirm(username)
	return nil
}

// SetMinimumConfirmationCodeLength sets the minimum length of the user
// confirmation code. The default is 20.
func (state *UserState) SetMinimumConfirmationCodeLength(length int) {
	minConfirmationCodeLength = length
}

// GenerateUniqueConfirmationCode generates a unique confirmation code that
// can be used for confirming users.
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
			return confirmationCode, errors.New("Too many generated confirmation codes are not unique!")
		}
	}
	return confirmationCode, nil
}

// ValidUsernamePassword only checks if the given username and password are
// different and if they only contain letters, numbers and/or underscore.
// For checking if a given password is correct, use the `CorrectPassword`
// function instead.
func ValidUsernamePassword(username, password string) error {
	const allowedLetters = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
NEXT:
	for _, letter := range username {
		for _, allowedLetter := range allowedLetters {
			if letter == allowedLetter {
				continue NEXT // check the next letter in the username
			}
		}
		return errors.New("Only letters, numbers and underscore are allowed in usernames.")
	}
	if username == password {
		return errors.New("Username and password must be different, try another password.")
	}
	return nil
}

// Users returns a hash map of all the users.
func (state *UserState) Users() pinterface.IHashMap {
	return state.users
}

// Creator returns a struct for creating data structures.
func (state *UserState) Creator() pinterface.ICreator {
	return simplebolt.NewCreator(state.db)
}
