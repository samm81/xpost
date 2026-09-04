package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRootNoArgsPrintsHelp(t *testing.T) {
	messageFlag = ""
	linkFlag = ""
	imagePath = ""
	imageAlt = ""
	targetsFlag = nil
	configPath = ""
	dryRun = false
	verbose = false

	root := newRootCommand()
	root.SetIn(strings.NewReader(""))
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(nil)

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Usage:") || !strings.Contains(output.String(), "xpost publishes the same update") {
		t.Fatalf("output = %q, want help", output.String())
	}
}

func TestResolveMessageReadsPipedInput(t *testing.T) {
	messageFlag = ""
	linkFlag = ""
	imagePath = ""
	imageAlt = ""
	targetsFlag = nil
	configPath = ""
	dryRun = false
	verbose = false

	root := newRootCommand()
	root.SetIn(strings.NewReader("hello from stdin\n"))
	message, err := resolveMessage(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if message != "hello from stdin" {
		t.Fatalf("message = %q, want hello from stdin", message)
	}
}

func TestBuildPostersReportsMissingConfiguration(t *testing.T) {
	for _, variable := range []string{
		"XPOST_BLUESKY_HANDLE",
		"XPOST_BLUESKY_APP_PASSWORD",
		"XPOST_BLUESKY_PDS_URL",
		"XPOST_MASTODON_SERVER",
		"XPOST_MASTODON_ACCESS_TOKEN",
		"XPOST_TWITTER_CONSUMER_KEY",
		"XPOST_TWITTER_CONSUMER_SECRET",
		"XPOST_TWITTER_ACCESS_TOKEN",
		"XPOST_TWITTER_ACCESS_TOKEN_SECRET",
	} {
		t.Setenv(variable, "")
	}

	_, err := buildPosters(context.Background(), []string{"bluesky", "mastodon", "twitter"}, configuration{})
	if err == nil {
		t.Fatal("buildPosters() error = nil, want configuration error")
	}
	var configurationErr configurationError
	if !errors.As(err, &configurationErr) {
		t.Fatalf("error = %T, want configurationError", err)
	}
	message := err.Error()
	for _, expected := range []string{
		"no publishing targets are configured",
		"bluesky: XPOST_BLUESKY_HANDLE, XPOST_BLUESKY_APP_PASSWORD",
		"mastodon: XPOST_MASTODON_SERVER, XPOST_MASTODON_ACCESS_TOKEN",
		"twitter: XPOST_TWITTER_CONSUMER_KEY",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error = %q, want %q", message, expected)
		}
	}
}
