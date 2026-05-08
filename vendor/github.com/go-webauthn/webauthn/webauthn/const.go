package webauthn

import (
	"time"
)

const (
	errFmtFieldNotValidDomainString = "field '%s' is not a valid domain string: %w"
	errFmtConfigValidate            = "error occurred validating the configuration: %w"
)

const (
	defaultTimeoutUVD = time.Millisecond * 120000
	defaultTimeout    = time.Millisecond * 300000
)
