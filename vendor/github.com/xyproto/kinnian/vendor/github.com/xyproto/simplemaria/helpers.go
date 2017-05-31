package simplemaria

import (
	"log"
	"strconv"
	"strings"
)

var Verbose = false

/* --- Helper functions --- */

// Split a string into two parts, given a delimiter.
// Returns the two parts and true if it works out.
func twoFields(s, delim string) (string, string, bool) {
	if strings.Count(s, delim) != 1 {
		return s, "", false
	}
	fields := strings.Split(s, delim)
	return fields[0], fields[1], true
}

func leftOf(s, delim string) string {
	if left, _, ok := twoFields(s, delim); ok {
		return strings.TrimSpace(left)
	}
	return ""
}

func rightOf(s, delim string) string {
	if _, right, ok := twoFields(s, delim); ok {
		return strings.TrimSpace(right)
	}
	return ""
}

// Parse a DSN
func splitConnectionString(connectionString string) (string, string, bool, string, string, string) {
	var (
		userPass, hostPortDatabase, dbname       string
		hostPort, password, username, port, host string
		hasPassword                              bool
	)

	// Gather the fields

	// Optional left side of @ with username and password
	userPass = leftOf(connectionString, "@")
	if userPass != "" {
		hostPortDatabase = rightOf(connectionString, "@")
	} else {
		if strings.HasSuffix(connectionString, "@") {
			hostPortDatabase = connectionString[:len(connectionString)-1]
		} else {
			hostPortDatabase = connectionString
		}
	}
	// Optional right side of / with database name
	dbname = rightOf(hostPortDatabase, "/")
	if dbname != "" {
		hostPort = leftOf(hostPortDatabase, "/")
	} else {
		if strings.HasSuffix(hostPortDatabase, "/") {
			hostPort = hostPortDatabase[:len(hostPortDatabase)-1]
		} else {
			hostPort = hostPortDatabase
		}
		dbname = defaultDatabaseName
	}
	// Optional right side of : with password
	password = rightOf(userPass, ":")
	if password != "" {
		username = leftOf(userPass, ":")
	} else {
		if strings.HasSuffix(userPass, ":") {
			username = userPass[:len(userPass)-1]
			hasPassword = true
		} else {
			username = userPass
		}
	}
	// Optional right side of : with port
	port = rightOf(hostPort, ":")
	if port != "" {
		host = leftOf(hostPort, ":")
	} else {
		if strings.HasSuffix(hostPort, ":") {
			host = hostPort[:len(hostPort)-1]
		} else {
			host = hostPort
		}
		if host != "" {
			port = strconv.Itoa(defaultPort)
		}
	}

	if Verbose {
		log.Println("Connection:")
		log.Println("\tusername:\t", username)
		log.Println("\tpassword:\t", password)
		log.Println("\thas password:\t", hasPassword)
		log.Println("\thost:\t\t", host)
		log.Println("\tport:\t\t", port)
		log.Println("\tdbname:\t\t", dbname)
		log.Println()
	}

	return username, password, hasPassword, host, port, dbname
}

// Build a DSN
func buildConnectionString(username, password string, hasPassword bool, host, port, dbname string) string {

	// Build the new connection string

	newConnectionString := ""
	if (host != "") && (port != "") {
		newConnectionString += "tcp(" + host + ":" + port + ")"
	} else if host != "" {
		newConnectionString += "tcp(" + host + ")"
	} else if port != "" {
		newConnectionString += "tcp(" + ":" + port + ")"
		log.Fatalln("There is only a port. This should not happen.")
	}
	if (username != "") && hasPassword {
		newConnectionString = username + ":" + password + "@" + newConnectionString
	} else if username != "" {
		newConnectionString = username + "@" + newConnectionString
	} else if hasPassword {
		newConnectionString = ":" + password + "@" + newConnectionString
	}
	newConnectionString += "/"

	if Verbose {
		log.Println("DSN:", newConnectionString)
	}

	return newConnectionString
}

// Take apart and rebuild the connection string. Also return the dbname.
func rebuildConnectionString(connectionString string) (string, string) {
	username, password, hasPassword, hostname, port, dbname := splitConnectionString(connectionString)
	return buildConnectionString(username, password, hasPassword, hostname, port, dbname), dbname
}
