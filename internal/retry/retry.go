// Package retry provides bounded retries for transient provider operations.
package retry

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/blacktop/xpost/internal/logutil"
)

const (
	defaultAttempts    = 3
	defaultInitialWait = 500 * time.Millisecond
	defaultMaximumWait = 4 * time.Second
)

// Policy controls the number and spacing of retry attempts.
type Policy struct {
	Attempts    int
	InitialWait time.Duration
	MaximumWait time.Duration
}

// DefaultPolicy returns the retry policy used for provider network operations.
func DefaultPolicy() Policy {
	return Policy{
		Attempts:    defaultAttempts,
		InitialWait: defaultInitialWait,
		MaximumWait: defaultMaximumWait,
	}
}

// Do calls operation until it succeeds, returns a non-transient error, or
// exhausts the policy. The operation is never retried after context cancellation.
func Do(
	ctx context.Context,
	policy Policy,
	operation string,
	transient func(error) bool,
	call func() error,
) error {
	policy = normalizePolicy(policy)

	var lastErr error

	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = call()
		if lastErr == nil {
			return nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		if attempt == policy.Attempts || transient == nil || !transient(lastErr) {
			return lastErr
		}

		wait := backoff(policy, attempt)
		logutil.Infof("retrying %s (attempt %d/%d) in %s", operation, attempt+1, policy.Attempts, wait)

		if err := waitContext(ctx, wait); err != nil {
			return err
		}
	}

	return lastErr
}

// HTTPStatus reports whether an HTTP status normally indicates a transient
// server, timeout, or rate-limit condition.
func HTTPStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooEarly ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError &&
			status < 600
}

// NetworkError reports whether err is a retryable network or incomplete-response
// error. Context cancellation and deadline errors are handled by the caller.
func NetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return true
	}

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

func normalizePolicy(policy Policy) Policy {
	if policy.Attempts <= 0 {
		policy.Attempts = defaultAttempts
	}

	if policy.InitialWait < 0 {
		policy.InitialWait = defaultInitialWait
	}

	if policy.MaximumWait < 0 {
		policy.MaximumWait = defaultMaximumWait
	}

	if policy.MaximumWait > 0 && policy.InitialWait > policy.MaximumWait {
		policy.InitialWait = policy.MaximumWait
	}

	return policy
}

func backoff(policy Policy, attempt int) time.Duration {
	wait := policy.InitialWait
	for step := 1; step < attempt && wait > 0; step++ {
		if policy.MaximumWait > 0 && wait >= policy.MaximumWait/2 {
			return policy.MaximumWait
		}

		wait *= 2
	}

	if policy.MaximumWait > 0 && wait > policy.MaximumWait {
		return policy.MaximumWait
	}

	return wait
}

func waitContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
