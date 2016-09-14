package permissionsql

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/xyproto/cookie"         // For cookies
	"github.com/xyproto/pinterface"     // For interfaces
	db "github.com/xyproto/simplemaria" // MariaDB/MySQL database wrapper
)

const (
	// username:password@host:port/database
	defaultConnectionString = "localhost:3306/"
)

var (
	minConfirmationCodeLength = 20 // minimum length of the confirmation code
)

type UserState struct {
	users             *db.HashMap // Hash map of users, with several different fields per user ("loggedin", "confirmed", "email" etc)
	usernames         *db.Set     // A list of all usernames, for easy enumeration
	unconfirmed       *db.Set     // A list of unconfirmed usernames, for easy enumeration
	host              *db.Host    // A database host
	cookieSecret      string      // Secret for storing secure cookies
	cookieTime        int64       // How long a cookie should last, in seconds
	passwordAlgorithm string      // The hashing algorithm to utilize default: "bcrypt+" allowed: ("sha256", "bcrypt", "bcrypt+")
}

// Create a new *UserState that can be used for managing users.
// The random number generator will be seeded after generating the cookie secret.
// A Host* for the local MariaDB/MySQL server will be created.
func NewUserStateSimple() (*UserState, error) {
	// connection string | initialize random generator after generating the cookie secret
	return NewUserState(defaultConnectionString, true)
}

// Create a new *UserState that can be used for managing users.
// connectionString may be on the form "username:password@host:port/database".
// If randomseed is true, the random number generator will be seeded after generating the cookie secret (true is a good default value).
func NewUserStateWithDSN(connectionString string, database_name string, randomseed bool) (*UserState, error) {
	// Test connection
	if err := db.TestConnectionHostWithDSN(connectionString); err != nil {
		return nil, err
	}

	host := db.NewHostWithDSN(connectionString, database_name)

	state := new(UserState)

	var err error
	state.users, err = db.NewHashMap(host, "users")
	if err != nil {
		return nil, err
	}
	state.usernames, err = db.NewSet(host, "usernames")
	if err != nil {
		return nil, err
	}
	state.unconfirmed, err = db.NewSet(host, "unconfirmed")
	if err != nil {
		return nil, err
	}

	state.host = host

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

	if err := host.Ping(); err != nil {
		defer host.Close()
		return nil, errors.New(fmt.Sprintf("Error when pinging %s: %s\n", connectionString, err.Error()))
	}

	return state, nil
}

// Create a new *UserState that can be used for managing users.
// connectionString may be on the form "username:password@host:port/database".
// If randomseed is true, the random number generator will be seeded after generating the cookie secret (true is a good default value).
func NewUserState(connectionString string, randomseed bool) (*UserState, error) {
	// Test connection
	if err := db.TestConnectionHost(connectionString); err != nil {
		return nil, err
	}

	host := db.NewHost(connectionString)

	state := new(UserState)

	var err error
	state.users, err = db.NewHashMap(host, "users")
	if err != nil {
		return nil, err
	}
	state.usernames, err = db.NewSet(host, "usernames")
	if err != nil {
		return nil, err
	}
	state.unconfirmed, err = db.NewSet(host, "unconfirmed")
	if err != nil {
		return nil, err
	}

	state.host = host

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

	if err := host.Ping(); err != nil {
		defer host.Close()
		return nil, errors.New(fmt.Sprintf("Error when pinging %s: %s\n", connectionString, err.Error()))
	}

	return state, nil
}

// Get the database host
func (state *UserState) Host() pinterface.IHost {
	return state.host
}

// Close the connection to the database host
func (state *UserState) Close() {
	state.host.Close()
}

// Check if the current user is logged in and has user rights.
func (state *UserState) UserRights(req *http.Request) bool {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return false
	}
	return state.IsLoggedIn(username)
}

// Check if the given username exists.
func (state *UserState) HasUser(username string) bool {
	val, err := state.usernames.Has(username)
	if err != nil {
		// This happened at concurrent connections before introducing the connection pool
		panic("ERROR: Lost connection to database?")
	}
	return val
}

// Return the boolean value for a given username and fieldname.
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

// Store a boolean value for the given username and custom fieldname.
func (state *UserState) SetBooleanField(username, fieldname string, val bool) {
	strval := "false"
	if val {
		strval = "true"
	}
	state.users.Set(username, fieldname, strval)
}

// Check if the given username is confirmed.
func (state *UserState) IsConfirmed(username string) bool {
	return state.BooleanField(username, "confirmed")
}

// Checks if the given username is logged in.
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

// Check if the current user is logged in and has administrator rights.
func (state *UserState) AdminRights(req *http.Request) bool {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return false
	}
	return state.IsLoggedIn(username) && state.IsAdmin(username)
}

// Check if the given username is an administrator.
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

// Retrieve the username that is stored in a cookie in the browser, if available.
func (state *UserState) UsernameCookie(req *http.Request) (string, error) {
	username, ok := cookie.SecureCookie(req, "user", state.cookieSecret)
	if ok && (username != "") {
		return username, nil
	}
	return "", errors.New("Could not retrieve the username from browser cookie")
}

// Store the given username in a cookie in the browser, if possible.
// The user must exist.
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

// Get a list of all usernames.
func (state *UserState) AllUsernames() ([]string, error) {
	return state.usernames.GetAll()
}

// Get the email for the given username.
func (state *UserState) Email(username string) (string, error) {
	return state.users.Get(username, "email")
}

// Get the password hash for the given username.
func (state *UserState) PasswordHash(username string) (string, error) {
	return state.users.Get(username, "password")
}

// Get all registered users that are not yet confirmed.
func (state *UserState) AllUnconfirmedUsernames() ([]string, error) {
	return state.unconfirmed.GetAll()
}

// Get the confirmation code for a specific user.
func (state *UserState) ConfirmationCode(username string) (string, error) {
	return state.users.Get(username, "confirmationCode")
}

// Get the users HashMap.
func (state *UserState) Users() pinterface.IHashMap {
	return state.users
}

// Add a user that is registered but not confirmed.
func (state *UserState) AddUnconfirmed(username, confirmationCode string) {
	state.unconfirmed.Add(username)
	state.users.Set(username, "confirmationCode", confirmationCode)
}

// Remove a user that is registered but not confirmed.
func (state *UserState) RemoveUnconfirmed(username string) {
	state.unconfirmed.Del(username)
	state.users.DelKey(username, "confirmationCode")
}

// Mark a user as confirmed.
func (state *UserState) MarkConfirmed(username string) {
	state.users.Set(username, "confirmed", "true")
}

// Remove user and login status.
func (state *UserState) RemoveUser(username string) {
	state.usernames.Del(username)
	// Remove additional data as well
	//state.users.DelKey(username, "loggedin")
	state.users.Del(username)
}

// Mark user as an administrator.
func (state *UserState) SetAdminStatus(username string) {
	state.users.Set(username, "admin", "true")
}

// Mark user as a regular user.
func (state *UserState) RemoveAdminStatus(username string) {
	state.users.Set(username, "admin", "false")
}

// Creates a user from the username and password hash, does not check for rights.
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

// Creates a user and hashes the password, does not check for rights.
// The given data must be valid.
func (state *UserState) AddUser(username, password, email string) {
	passwordHash := state.HashPassword(username, password)
	state.addUserUnchecked(username, passwordHash, email)
}

// Mark the user as logged in. Use the Login function instead, unless cookies are not involved.
func (state *UserState) SetLoggedIn(username string) {
	state.users.Set(username, "loggedin", "true")
}

// Mark the user as logged out.
func (state *UserState) SetLoggedOut(username string) {
	state.users.Set(username, "loggedin", "false")
}

// Convenience function for logging a user in and storing the username in a cookie.
// Returns an error if the cookie could not be set.
func (state *UserState) Login(w http.ResponseWriter, username string) error {
	state.SetLoggedIn(username)
	return state.SetUsernameCookie(w, username)
}

// Try to clear the user cookie by setting it to expired.
// Some browsers *may* be configured to keep cookies even after this.
func (state *UserState) ClearCookie(w http.ResponseWriter) {
	cookie.ClearCookie(w, "user", "/")
}

// Convenience function for logging a user out.
func (state *UserState) Logout(username string) {
	state.SetLoggedOut(username)
}

// Convenience function that will return a username (from the browser cookie) or an empty string.
func (state *UserState) Username(req *http.Request) string {
	username, err := state.UsernameCookie(req)
	if err != nil {
		return ""
	}
	return username
}

// Get how long a login cookie should last, in seconds.
func (state *UserState) CookieTimeout(username string) int64 {
	return state.cookieTime
}

// Set how long a login cookie should last, in seconds.
func (state *UserState) SetCookieTimeout(cookieTime int64) {
	state.cookieTime = cookieTime
}

// Get the current password hashing algorithm.
func (state *UserState) PasswordAlgo() string {
	return state.passwordAlgorithm
}

// Set the password hashing algorithm that should be used.
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

// Hash the password (takes a username as well, it can be used for salting).
func (state *UserState) HashPassword(username, password string) string {
	switch state.passwordAlgorithm {
	case "sha256":
		return string(hash_sha256(state.cookieSecret, username, password))
	case "bcrypt", "bcrypt+":
		return string(hash_bcrypt(password))
	}
	// Only valid password algorithms should be allowed to set
	return ""
}

// Return the stored hash, or an empty byte slice.
func (state *UserState) stored_hash(username string) []byte {
	hashString, err := state.PasswordHash(username)
	if err != nil {
		return []byte{}
	}
	return []byte(hashString)
}

// Check if a password is correct. username is needed because it is part of the hash.
func (state *UserState) CorrectPassword(username, password string) bool {

	if !state.HasUser(username) {
		return false
	}

	// Retrieve the stored password hash
	hash := state.stored_hash(username)
	if len(hash) == 0 {
		return false
	}

	// Check the password with the right password algorithm
	switch state.passwordAlgorithm {
	case "sha256":
		return correct_sha256(hash, state.cookieSecret, username, password)
	case "bcrypt":
		return correct_bcrypt(hash, password)
	case "bcrypt+": // for backwards compatibility with sha256
		if is_sha256(hash) && correct_sha256(hash, state.cookieSecret, username, password) {
			return true
		} else {
			return correct_bcrypt(hash, password)
		}
	}
	return false
}

// Goes through all the confirmationCodes of all the unconfirmed users
// and checks if this confirmationCode already is in use.
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

// Given a unique confirmation code, find the corresponding username.
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

// Remove the username from the list of unconfirmed users and mark the user as confirmed.
func (state *UserState) Confirm(username string) {
	// Remove from the list of unconfirmed usernames
	state.RemoveUnconfirmed(username)

	// Mark user as confirmed
	state.MarkConfirmed(username)
}

// Take a confirmation code and mark the corresponding unconfirmed user as confirmed.
func (state *UserState) ConfirmUserByConfirmationCode(confirmationcode string) error {
	if username, err := state.FindUserByConfirmationCode(confirmationcode); err != nil {
		return err
	} else {
		state.Confirm(username)
	}
	return nil
}

// Set the minimum length of the user confirmation code. The default is 20.
func (state *UserState) SetMinimumConfirmationCodeLength(length int) {
	minConfirmationCodeLength = length
}

// Generate a unique confirmation code that can be used for confirming users.
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

// Check that the given username and password are different.
// Also check if the chosen username only contains letters, numbers and/or underscore.
// Use the "CorrectPassword" function for checking if the password is correct.
func ValidUsernamePassword(username, password string) error {
	const allowed_letters = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
NEXT:
	for _, letter := range username {
		for _, allowedLetter := range allowed_letters {
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

// Return a struct for creating datastructures
func (state *UserState) Creator() pinterface.ICreator {
	return db.NewCreator(state.host)
}
