// Package config provides data structure to configure rate-limiter.
package config

import (
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

// NewLimiter is a constructor for Limiter.
func NewLimiter(max int64, ttl time.Duration) *Limiter {
	limiter := &Limiter{Max: max, TTL: ttl}
	limiter.MessageContentType = "text/plain; charset=utf-8"
	limiter.Message = "You have reached maximum request limit."
	limiter.StatusCode = 429
	limiter.IPLookups = []string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}

	limiter.tokenBucketsNoTTL = make(map[string]*rate.Limiter)

	return limiter
}

// NewLimiterExpiringBuckets constructs Limiter with expirable TokenBuckets.
func NewLimiterExpiringBuckets(max int64, ttl, bucketDefaultExpirationTTL, bucketExpireJobInterval time.Duration) *Limiter {
	limiter := NewLimiter(max, ttl)
	limiter.TokenBuckets.DefaultExpirationTTL = bucketDefaultExpirationTTL
	limiter.TokenBuckets.ExpireJobInterval = bucketExpireJobInterval

	// Default for ExpireJobInterval is every minute.
	if limiter.TokenBuckets.ExpireJobInterval <= 0 {
		limiter.TokenBuckets.ExpireJobInterval = time.Minute
	}

	limiter.tokenBucketsWithTTL = gocache.New(
		limiter.TokenBuckets.DefaultExpirationTTL,
		limiter.TokenBuckets.ExpireJobInterval,
	)

	return limiter
}

// Limiter is a config struct to limit a particular request handler.
type Limiter struct {
	// HTTP message when limit is reached.
	Message string

	// Content-Type for Message
	MessageContentType string

	// HTTP status code when limit is reached.
	StatusCode int

	// Maximum number of requests to limit per duration.
	Max int64

	// Duration of rate-limiter.
	TTL time.Duration

	// List of places to look up IP address.
	// Default is "RemoteAddr", "X-Forwarded-For", "X-Real-IP".
	// You can rearrange the order as you like.
	IPLookups []string

	// List of HTTP Methods to limit (GET, POST, PUT, etc.).
	// Empty means limit all methods.
	Methods []string

	// List of HTTP headers to limit.
	// Empty means skip headers checking.
	Headers map[string][]string

	// List of basic auth usernames to limit.
	BasicAuthUsers []string

	// Able to configure token bucket expirations.
	TokenBuckets struct {
		// Default TTL to expire bucket per key basis.
		DefaultExpirationTTL time.Duration

		// How frequently tollbooth will trigger the expire job
		ExpireJobInterval time.Duration
	}

	// Map of limiters without TTL
	tokenBucketsNoTTL map[string]*rate.Limiter

	// Map of limiters with TTL
	tokenBucketsWithTTL *gocache.Cache

	sync.RWMutex
}

func (l *Limiter) isUsingTokenBucketsWithTTL() bool {
	return l.TokenBuckets.DefaultExpirationTTL > 0
}

func (l *Limiter) limitReachedNoTokenBucketTTL(key string) bool {
	l.Lock()
	defer l.Unlock()

	if _, found := l.tokenBucketsNoTTL[key]; !found {
		l.tokenBucketsNoTTL[key] = rate.NewLimiter(rate.Every(l.TTL), int(l.Max))
	}

	return !l.tokenBucketsNoTTL[key].AllowN(time.Now(), 1)
}

func (l *Limiter) limitReachedWithDefaultTokenBucketTTL(key string) bool {
	return l.limitReachedWithCustomTokenBucketTTL(key, gocache.DefaultExpiration)
}

func (l *Limiter) limitReachedWithCustomTokenBucketTTL(key string, tokenBucketTTL time.Duration) bool {
	l.Lock()
	defer l.Unlock()

	if _, found := l.tokenBucketsWithTTL.Get(key); !found {
		l.tokenBucketsWithTTL.Set(
			key,
			rate.NewLimiter(rate.Every(l.TTL), int(l.Max)),
			tokenBucketTTL,
		)
	}

	expiringMap, found := l.tokenBucketsWithTTL.Get(key)
	if !found {
		return false
	}

	return !expiringMap.(*rate.Limiter).AllowN(time.Now(), 1)
}

// LimitReached returns a bool indicating if the Bucket identified by key ran out of tokens.
func (l *Limiter) LimitReached(key string) bool {
	if l.isUsingTokenBucketsWithTTL() {
		return l.limitReachedWithDefaultTokenBucketTTL(key)

	} else {
		return l.limitReachedNoTokenBucketTTL(key)
	}

	return false
}

// LimitReachedWithCustomTokenBucketTTL returns a bool indicating if the Bucket identified by key ran out of tokens.
// This public API allows user to define custom expiration TTL on the key.
func (l *Limiter) LimitReachedWithCustomTokenBucketTTL(key string, tokenBucketTTL time.Duration) bool {
	if l.isUsingTokenBucketsWithTTL() {
		return l.limitReachedWithCustomTokenBucketTTL(key, tokenBucketTTL)

	} else {
		return l.limitReachedNoTokenBucketTTL(key)
	}

	return false
}
