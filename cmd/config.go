package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const defaultConfigFileName = "config.toml"

type configuration struct {
	Bluesky  blueskyConfiguration  `toml:"bluesky"`
	Mastodon mastodonConfiguration `toml:"mastodon"`
	Twitter  twitterConfiguration  `toml:"twitter"`
}

type blueskyConfiguration struct {
	Handle      string `toml:"handle"`
	AppPassword string `toml:"app_password"`
	PDSURL      string `toml:"pds_url"`
}

type mastodonConfiguration struct {
	Server       string `toml:"server"`
	AccessToken  string `toml:"access_token"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

type twitterConfiguration struct {
	ConsumerKey       string `toml:"consumer_key"`
	ConsumerSecret    string `toml:"consumer_secret"`
	AccessToken       string `toml:"access_token"`
	AccessTokenSecret string `toml:"access_token_secret"`
}

type configLoadError struct {
	path string
	err  error
}

func (e configLoadError) Error() string {
	if e.path == "" {
		return fmt.Sprintf("load xpost config: %v", e.err)
	}
	return fmt.Sprintf("load xpost config %q: %v", e.path, e.err)
}

func (e configLoadError) Unwrap() error {
	return e.err
}

func loadConfiguration(path string) (configuration, error) {
	path = strings.TrimSpace(path)
	explicitPath := path != ""
	if !explicitPath {
		configDirectory, err := os.UserConfigDir()
		if err != nil {
			return configuration{}, configLoadError{
				err: fmt.Errorf("resolve default config path: %w", err),
			}
		}
		path = filepath.Join(configDirectory, "xpost", defaultConfigFileName)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !explicitPath && errors.Is(err, os.ErrNotExist) {
			loaded := configuration{}
			loaded.applyEnvironment()
			return loaded, nil
		}
		return configuration{}, configLoadError{
			path: path,
			err:  fmt.Errorf("read file: %w", err),
		}
	}

	var loaded configuration
	if err := toml.NewDecoder(bytes.NewReader(data)).
		DisallowUnknownFields().
		Decode(&loaded); err != nil {
		return configuration{}, configLoadError{
			path: path,
			err:  fmt.Errorf("parse file: %w", err),
		}
	}

	loaded.applyEnvironment()
	return loaded, nil
}

func (c *configuration) applyEnvironment() {
	c.Bluesky.Handle = configValue(c.Bluesky.Handle, "XPOST_BLUESKY_HANDLE")
	c.Bluesky.AppPassword = configValue(c.Bluesky.AppPassword, "XPOST_BLUESKY_APP_PASSWORD")
	c.Bluesky.PDSURL = configValue(c.Bluesky.PDSURL, "XPOST_BLUESKY_PDS_URL")

	c.Mastodon.Server = configValue(c.Mastodon.Server, "XPOST_MASTODON_SERVER")
	c.Mastodon.AccessToken = configValue(c.Mastodon.AccessToken, "XPOST_MASTODON_ACCESS_TOKEN")
	c.Mastodon.ClientID = configValue(c.Mastodon.ClientID, "XPOST_MASTODON_CLIENT_ID")
	c.Mastodon.ClientSecret = configValue(c.Mastodon.ClientSecret, "XPOST_MASTODON_CLIENT_SECRET")

	c.Twitter.ConsumerKey = configValue(c.Twitter.ConsumerKey, "XPOST_TWITTER_CONSUMER_KEY")
	c.Twitter.ConsumerSecret = configValue(c.Twitter.ConsumerSecret, "XPOST_TWITTER_CONSUMER_SECRET")
	c.Twitter.AccessToken = configValue(c.Twitter.AccessToken, "XPOST_TWITTER_ACCESS_TOKEN")
	c.Twitter.AccessTokenSecret = configValue(c.Twitter.AccessTokenSecret, "XPOST_TWITTER_ACCESS_TOKEN_SECRET")
}

func configValue(fileValue, environmentName string) string {
	if value := strings.TrimSpace(os.Getenv(environmentName)); value != "" {
		return value
	}
	return strings.TrimSpace(fileValue)
}
