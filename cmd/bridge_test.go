package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestBridgeValidateDoesNotLogIn(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.toml")
	config := `[bluesky]
handle = "person.example"
app_password = "app-password"
pds_url = "http://127.0.0.1:1"
`
	if err := os.WriteFile(configFile, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	previousConfigPath := configPath
	configPath = configFile
	t.Cleanup(func() { configPath = previousConfigPath })

	command := newBridgeCommand()
	command.SetIn(strings.NewReader(`{"operation":"validate","target":"bluesky","text":"hello"}`))
	var output bytes.Buffer
	command.SetOut(&output)

	if err := runBridge(command, nil); err != nil {
		t.Fatal(err)
	}

	var response xpost.BridgeResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("response = %q: %v", output.String(), err)
	}
	if response.Status != xpost.BridgeStatusValidated {
		t.Fatalf("status = %q, want %q", response.Status, xpost.BridgeStatusValidated)
	}
}
