package cmd

import (
	"testing"

	"github.com/blacktop/xpost/internal/xpost"
)

func TestBridgeErrorResponseClassifiesConfiguration(t *testing.T) {
	t.Parallel()

	response := bridgeErrorResponse(newConfigurationError([]targetConfiguration{
		{target: "bluesky", variables: []string{"XPOST_BLUESKY_HANDLE"}},
	}))
	if response.Status != xpost.BridgeStatusRejected {
		t.Fatalf("status = %q, want %q", response.Status, xpost.BridgeStatusRejected)
	}
	if response.ErrorKind != "configuration" {
		t.Fatalf("error kind = %q, want configuration", response.ErrorKind)
	}
}

func TestBridgeErrorResponseClassifiesConfigLoadFailure(t *testing.T) {
	t.Parallel()

	response := bridgeErrorResponse(configLoadError{path: "/tmp/config.toml", err: errMessageRequired})
	if response.Status != xpost.BridgeStatusRejected {
		t.Fatalf("status = %q, want %q", response.Status, xpost.BridgeStatusRejected)
	}
	if response.ErrorKind != "configuration" {
		t.Fatalf("error kind = %q, want configuration", response.ErrorKind)
	}
}
