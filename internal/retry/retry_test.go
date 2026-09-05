package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDoRetriesTransientErrors(t *testing.T) {
	t.Parallel()

	var calls int

	err := Do(context.Background(), Policy{Attempts: 3}, "test operation", func(error) bool {
		return true
	}, func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}

	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestDoStopsOnNonTransientError(t *testing.T) {
	t.Parallel()

	var calls int

	wantErr := errors.New("permanent")

	err := Do(context.Background(), Policy{Attempts: 3}, "test operation", func(error) bool {
		return false
	}, func() error {
		calls++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Do() error = %v, want %v", err, wantErr)
	}

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int

	err := Do(ctx, Policy{Attempts: 3, InitialWait: time.Hour}, "test operation", func(error) bool {
		return true
	}, func() error {
		calls++

		cancel()

		return errors.New("temporary")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context canceled", err)
	}

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestHTTPStatus(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		status int
		want   bool
	}{
		{name: "request timeout", status: http.StatusRequestTimeout, want: true},
		{name: "too early", status: http.StatusTooEarly, want: true},
		{name: "rate limited", status: http.StatusTooManyRequests, want: true},
		{name: "server error", status: http.StatusBadGateway, want: true},
		{name: "bad request", status: http.StatusBadRequest, want: false},
		{name: "forbidden", status: http.StatusForbidden, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := HTTPStatus(test.status); got != test.want {
				t.Fatalf("HTTPStatus(%d) = %t, want %t", test.status, got, test.want)
			}
		})
	}
}
