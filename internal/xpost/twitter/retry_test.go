package twitter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/blacktop/xpost/internal/retry"
	"github.com/blacktop/xpost/internal/xpost"
	"github.com/michimani/gotwi"
)

func TestPostResultRetriesTransientHTTPError(t *testing.T) {
	t.Parallel()

	var calls int
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.String() != "https://api.twitter.com/2/tweets" {
				return nil, fmt.Errorf("unexpected request URL: %s", req.URL)
			}
			if calls == 1 {
				return response(req, http.StatusServiceUnavailable, `{"title":"temporary"}`), nil
			}
			return response(req, http.StatusCreated, `{"data":{"id":"123"}}`), nil
		}),
	}
	api, err := gotwi.NewClient(&gotwi.NewClientInput{
		HTTPClient:           httpClient,
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           "access-token",
		OAuthTokenSecret:     "access-secret",
		APIKey:               "consumer-key",
		APIKeySecret:         "consumer-secret",
	})
	if err != nil {
		t.Fatalf("gotwi.NewClient() error = %v", err)
	}

	client := &Client{
		api: api,
		retryPolicy: retry.Policy{
			Attempts:    3,
			InitialWait: 0,
		},
	}
	result, err := client.PostResult(context.Background(), xpost.Request{Message: "hello"})
	if err != nil {
		t.Fatalf("PostResult() error = %v", err)
	}
	if result.RemoteID != "123" {
		t.Fatalf("PostResult() remote id = %q, want %q", result.RemoteID, "123")
	}
	if calls != 2 {
		t.Fatalf("request calls = %d, want 2", calls)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}
