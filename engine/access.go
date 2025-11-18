package engine

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// CommonLogFormat returns a line with the data that is available at the start
// of a request handler. The log line is in NCSA format, the same log format
// used by Apache. Fields where data is not available are indicated by a "-".
// See also: https://en.wikipedia.org/wiki/Common_Log_Format
func (ac *Config) CommonLogFormat(req *http.Request, statusCode int, byteSize int64) string {
	username := "-"
	if ac.perm != nil {
		username = ac.perm.UserState().Username(req)
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	statusCodeString := "-"
	if statusCode > 0 {
		statusCodeString = strconv.Itoa(statusCode)
	}
	byteSizeString := "0"
	if byteSize > 0 {
		byteSizeString = strconv.FormatInt(byteSize, 10)
	}
	timestamp := strings.Replace(time.Now().Format("02/Jan/2006 15:04:05 -0700"), " ", ":", 1)
	return fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %s %s", host, username, timestamp, req.Method, req.RequestURI, req.Proto, statusCodeString, byteSizeString)
}

// CombinedLogFormat returns a line with the data that is available at the start
// of a request handler. The log line is in CLF, similar to the Common log format,
// but with two extra fields.
// See also: https://httpd.apache.org/docs/1.3/logs.html#combined
func (ac *Config) CombinedLogFormat(req *http.Request, statusCode int, byteSize int64) string {
	username := "-"
	if ac.perm != nil {
		username = ac.perm.UserState().Username(req)
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	statusCodeString := "-"
	if statusCode > 0 {
		statusCodeString = strconv.Itoa(statusCode)
	}
	byteSizeString := "0"
	if byteSize > 0 {
		byteSizeString = fmt.Sprintf("%d", byteSize)
	}
	timestamp := strings.Replace(time.Now().Format("02/Jan/2006 15:04:05 -0700"), " ", ":", 1)
	referer := req.Header.Get("Referer")
	userAgent := req.Header.Get("User-Agent")
	return fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %s %s \"%s\" \"%s\"", host, username, timestamp, req.Method, req.RequestURI, req.Proto, statusCodeString, byteSizeString, referer, userAgent)
}

// LogAccess creates one entry in the access log, given a http.Request,
// a HTTP status code and the amount of bytes that have been transferred.
func (ac *Config) LogAccess(req *http.Request, statusCode int, byteSize int64) {
	if ac.commonAccessLogFilename != "" {
		go func(filename string) {
			f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil { // success
				defer f.Close()
				_, err = f.WriteString(ac.CommonLogFormat(req, statusCode, byteSize) + "\n")
				if err != nil {
					logrus.Warnf("Can not write to %s: %s", filename, err)
				}
			} else {
				logrus.Warnf("Can not open %s: %s", filename, err)
			}
		}(ac.commonAccessLogFilename)
	}
	if ac.combinedAccessLogFilename != "" {
		go func(filename string) {
			f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil { // success
				defer f.Close()
				_, err = f.WriteString(ac.CombinedLogFormat(req, statusCode, byteSize) + "\n")
				if err != nil {
					logrus.Warnf("Can not write to %s: %s", filename, err)
				}
			} else {
				logrus.Warnf("Can not open %s: %s", filename, err)
			}
		}(ac.combinedAccessLogFilename)
	}
}
