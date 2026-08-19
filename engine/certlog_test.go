package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// CertMagic messages must be logged like the rest of the messages, and not
// with a timestamp like "1.787175416960805e+09".
func TestCertMagicLoggerLogsWithLogrus(t *testing.T) {
	var buf bytes.Buffer

	standard := logrus.StandardLogger()
	output, formatter := standard.Out, standard.Formatter
	defer func() {
		logrus.SetOutput(output)
		logrus.SetFormatter(formatter)
	}()
	logrus.SetOutput(&buf)
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})

	certMagicLogger().Named("acme").Info("obtaining certificate", zap.String("identifier", "example.com"))

	line := buf.String()
	for _, substring := range []string{"level=info", "obtaining certificate", "identifier=example.com", "logger=acme"} {
		if !strings.Contains(line, substring) {
			t.Errorf("expected %q in %q", substring, line)
		}
	}
	if strings.Contains(line, "e+09") {
		t.Errorf("expected no floating point timestamp in %q", line)
	}
}

// Messages below the logrus log level must not be logged.
func TestCertMagicLoggerRespectsTheLogLevel(t *testing.T) {
	var buf bytes.Buffer

	standard := logrus.StandardLogger()
	output, formatter := standard.Out, standard.Formatter
	defer func() {
		logrus.SetOutput(output)
		logrus.SetFormatter(formatter)
	}()
	logrus.SetOutput(&buf)
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})

	certMagicLogger().Debug("checking certificate")

	if line := buf.String(); line != "" {
		t.Errorf("expected no output, got %q", line)
	}
}
