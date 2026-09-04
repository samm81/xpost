<p align="center">
  <a href="https://github.com/blacktop/xpost"><img alt="xpost Logo" src="https://raw.githubusercontent.com/blacktop/xpost/main/docs/logo.webp" height="200"/></a>
  <h4><p align="center">Cross post to all socials at once from your terminal</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/xpost/actions" alt="Actions">
          <img src="https://github.com/blacktop/xpost/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://github.com/blacktop/xpost/releases/latest" alt="Downloads">
          <img src="https://img.shields.io/github/downloads/blacktop/xpost/total.svg" /></a>
    <a href="https://github.com/blacktop/xpost/releases" alt="GitHub Release">
          <img src="https://img.shields.io/github/release/blacktop/xpost.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
<br>

## Supported Socials

- [x] X/Twitter
- [x] Mastodon
- [x] BlueSky 

## Getting Started

### Install

Via [homebrew](https://brew.sh)

```bash
brew install blacktop/tap/xpost
```

Via [Golang](https://go.dev/dl/)

```bash
go install github.com/blacktop/xpost@latest
```

Or download the latest [release](https://github.com/blacktop/xpost/releases/latest)

### Configuration

By default, xpost reads `~/.config/xpost/config.toml`. Use `--config` to
select another file. Keep this file private because it contains credentials.

```toml
[bluesky]
handle = "your.handle"
app_password = "your_app_password"
# pds_url = "https://bsky.social"

[mastodon]
server = "https://mastodon.social"
access_token = "your_token"
# client_id = "your_client_id"
# client_secret = "your_client_secret"

[twitter]
consumer_key = "your_key"
consumer_secret = "your_secret"
access_token = "your_token"
access_token_secret = "your_token_secret"
```

The environment variables from earlier versions are still supported and
override values from the TOML file. This is useful for temporary overrides and
existing scripts.

```bash
xpost --config ~/.config/xpost/config.toml "hello world"
```

### Usage

Send message to all supported networks

```bash
❱ xpost -m test --image docs/logo.webp
Posted to  Bluesky
Posted to  Mastodon
Posted to  Twitter/X
```

## License

MIT Copyright (c) 2025 **blacktop**
