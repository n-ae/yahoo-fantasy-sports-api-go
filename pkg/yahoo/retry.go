package yahoo

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Conservative default retry parameters for safe (GET) requests against
// transient failures. Override per-client with WithRetryPolicy.
const (
	defaultMaxRetries  = 3
	defaultBaseBackoff = 500 * time.Millisecond
	defaultMaxBackoff  = 8 * time.Second
)

// RetryPolicy controls how transient GET failures (429 and 5xx) are retried.
// The zero value is invalid; obtain a starting point from the defaults via
// NewClient and override with WithRetryPolicy.
type RetryPolicy struct {
	// MaxRetries is the number of retries after the first attempt (0 disables
	// retrying).
	MaxRetries int
	// BaseBackoff is the first backoff duration; it doubles each attempt.
	BaseBackoff time.Duration
	// MaxBackoff caps the backoff duration.
	MaxBackoff time.Duration
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:  defaultMaxRetries,
		BaseBackoff: defaultBaseBackoff,
		MaxBackoff:  defaultMaxBackoff,
	}
}

func (p RetryPolicy) validate() error {
	if p.MaxRetries < 0 {
		return fmt.Errorf("yahoo: RetryPolicy.MaxRetries must be >= 0")
	}
	if p.BaseBackoff <= 0 || p.MaxBackoff <= 0 {
		return fmt.Errorf("yahoo: RetryPolicy backoff durations must be > 0")
	}
	if p.MaxBackoff < p.BaseBackoff {
		return fmt.Errorf("yahoo: RetryPolicy.MaxBackoff must be >= BaseBackoff")
	}
	return nil
}

// isRetryableStatus reports whether an HTTP status warrants a bounded retry.
// GET requests are idempotent, so rate-limiting and transient server errors
// are safe to retry.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// backoffDelay returns an exponential backoff with full jitter for the given
// zero-based attempt, capped at the policy's MaxBackoff.
func backoffDelay(p RetryPolicy, attempt int) time.Duration {
	d := p.BaseBackoff << attempt
	if d > p.MaxBackoff || d <= 0 { // <=0 guards against shift overflow
		d = p.MaxBackoff
	}
	return time.Duration(rand.Int64N(int64(d) + 1))
}

// retryAfterDelay parses a Retry-After header value, which may be either a
// number of seconds or an HTTP date. It returns the delay and whether a valid
// value was found.
func retryAfterDelay(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(header); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// sleepCtx waits for d or until ctx is cancelled, returning ctx.Err() if the
// context is cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
