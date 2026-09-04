package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigurationReadsTOML(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "config.toml")
	configurationData := `[bluesky]
handle = "writer.example"
app_password = "bluesky-secret"
pds_url = "https://pds.example"

[mastodon]
server = "https://mastodon.example"
access_token = "mastodon-secret"
client_id = "mastodon-client"
client_secret = "mastodon-client-secret"

[twitter]
consumer_key = "twitter-key"
consumer_secret = "twitter-secret"
access_token = "twitter-token"
access_token_secret = "twitter-token-secret"
`
	configurationWriteFile(t, configurationPath, configurationData)
	clearConfigurationEnvironment(t)

	configuration, err := loadConfiguration(configurationPath)
	if err != nil {
		t.Fatal(err)
	}

	if configuration.Bluesky.Handle != "writer.example" ||
		configuration.Bluesky.AppPassword != "bluesky-secret" ||
		configuration.Bluesky.PDSURL != "https://pds.example" {
		t.Fatalf("bluesky configuration = %#v", configuration.Bluesky)
	}
	if configuration.Mastodon.Server != "https://mastodon.example" ||
		configuration.Mastodon.AccessToken != "mastodon-secret" ||
		configuration.Mastodon.ClientID != "mastodon-client" ||
		configuration.Mastodon.ClientSecret != "mastodon-client-secret" {
		t.Fatalf("mastodon configuration = %#v", configuration.Mastodon)
	}
	if configuration.Twitter.ConsumerKey != "twitter-key" ||
		configuration.Twitter.ConsumerSecret != "twitter-secret" ||
		configuration.Twitter.AccessToken != "twitter-token" ||
		configuration.Twitter.AccessTokenSecret != "twitter-token-secret" {
		t.Fatalf("twitter configuration = %#v", configuration.Twitter)
	}
}

func TestLoadConfigurationEnvironmentOverridesTOML(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "config.toml")
	configurationWriteFile(t, configurationPath, "[bluesky]\nhandle = \"file.example\"\n")
	clearConfigurationEnvironment(t)
	t.Setenv("XPOST_BLUESKY_HANDLE", "environment.example")

	configuration, err := loadConfiguration(configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bluesky.Handle != "environment.example" {
		t.Fatalf("handle = %q, want environment.example", configuration.Bluesky.Handle)
	}
}

func TestLoadConfigurationRejectsUnknownFields(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "config.toml")
	configurationWriteFile(t, configurationPath, "[bluesky]\nhandl = \"writer.example\"\n")

	_, err := loadConfiguration(configurationPath)
	if err == nil {
		t.Fatal("loadConfiguration() error = nil, want unknown-field error")
	}
}

func TestLoadConfigurationRequiresExplicitFile(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "missing.toml")

	_, err := loadConfiguration(configurationPath)
	if err == nil {
		t.Fatal("loadConfiguration() error = nil, want missing-file error")
	}
	if !strings.Contains(err.Error(), configurationPath) {
		t.Fatalf("error = %q, want path %q", err, configurationPath)
	}
}

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()
	for _, variable := range []string{
		"XPOST_BLUESKY_HANDLE",
		"XPOST_BLUESKY_APP_PASSWORD",
		"XPOST_BLUESKY_PDS_URL",
		"XPOST_MASTODON_SERVER",
		"XPOST_MASTODON_ACCESS_TOKEN",
		"XPOST_MASTODON_CLIENT_ID",
		"XPOST_MASTODON_CLIENT_SECRET",
		"XPOST_TWITTER_CONSUMER_KEY",
		"XPOST_TWITTER_CONSUMER_SECRET",
		"XPOST_TWITTER_ACCESS_TOKEN",
		"XPOST_TWITTER_ACCESS_TOKEN_SECRET",
	} {
		t.Setenv(variable, "")
	}
}

func configurationWriteFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
