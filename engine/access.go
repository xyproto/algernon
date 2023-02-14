package engine

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	ip := host
	if err != nil {
		ip = req.RemoteAddr
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
	return fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %s %s", ip, username, timestamp, req.Method, req.RequestURI, req.Proto, statusCodeString, byteSizeString)
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
	ip := host
	if err != nil {
		ip = req.RemoteAddr
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
	return fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %s %s \"%s\" \"%s\"", ip, username, timestamp, req.Method, req.RequestURI, req.Proto, statusCodeString, byteSizeString, referer, userAgent)
}

// LogAccess creates one entry in the access log, given a http.Request,
// a HTTP status code and the amount of bytes that have been transferred.
func (ac *Config) LogAccess(req *http.Request, statusCode int, byteSize int64) {
	if ac.commonAccessLogFilename != "" {
		f, err := os.OpenFile(ac.commonAccessLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Warnf("Can not open %s: %s", ac.commonAccessLogFilename, err)
			return
		}
		defer f.Close()
		_, err = f.WriteString(ac.CommonLogFormat(req, statusCode, byteSize) + "\n")
		if err != nil {
			log.Warnf("Can not write to %s: %s", ac.commonAccessLogFilename, err)
			return
		}
	}
	if ac.combinedAccessLogFilename != "" {
		f, err := os.OpenFile(ac.combinedAccessLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Warnf("Can not open %s: %s", ac.combinedAccessLogFilename, err)
			return
		}
		defer f.Close()
		_, err = f.WriteString(ac.CombinedLogFormat(req, statusCode, byteSize) + "\n")
		if err != nil {
			log.Warnf("Can not write to %s: %s", ac.combinedAccessLogFilename, err)
			return
		}
	}
}
