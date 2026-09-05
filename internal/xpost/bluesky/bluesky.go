package bluesky

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blacktop/xpost/internal/xpost"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/rivo/uniseg"
)

const (
	envHandle      = "XPOST_BLUESKY_HANDLE"
	envAppPassword = "XPOST_BLUESKY_APP_PASSWORD"
	envPDSURL      = "XPOST_BLUESKY_PDS_URL"

	providerName   = "bluesky"
	requestTimeout = 30 * time.Second
	maxGraphemes   = 300 // Bluesky's post character limit in graphemes
	maxBytes       = 3000
	maxImages      = 4
)

// Config contains Bluesky credentials and endpoint settings.
type Config struct {
	Handle      string
	AppPassword string
	PDSURL      string
}

// Client implements the xpost.Poster interface for Bluesky.
type Client struct {
	client *xrpc.Client
}

// New constructs a Bluesky poster.
func New(ctx context.Context, base Config) (xpost.Poster, error) {
	cfg, err := loadConfig(base)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: requestTimeout}
	userAgent := "xpost/1"
	xrpcClient := &xrpc.Client{
		Client:    httpClient,
		Host:      cfg.PDSURL,
		UserAgent: &userAgent,
	}

	session, err := atproto.ServerCreateSession(ctx, xrpcClient, &atproto.ServerCreateSession_Input{
		Identifier: cfg.Handle,
		Password:   cfg.AppPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	xrpcClient.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}

	return &Client{client: xrpcClient}, nil
}

// Name identifies the provider.
func (c *Client) Name() string { return providerName }

// Validate checks provider configuration and request constraints without network access.
func Validate(base Config, req xpost.Request) error {
	if _, err := loadConfig(base); err != nil {
		return err
	}

	return validateRequest(req)
}

// Validate checks if the request meets Bluesky's constraints.
func (c *Client) Validate(req xpost.Request) error {
	return validateRequest(req)
}

func requestText(req xpost.Request) string {
	if req.Link == "" {
		return req.Message
	}

	return req.Message + "\n\n" + req.Link
}

func validateRequest(req xpost.Request) error {
	if len(req.Attachments) > maxImages {
		return xpost.ValidationError{
			Provider: providerName,
			Reason:   fmt.Sprintf("too many images: %d (max %d)", len(req.Attachments), maxImages),
		}
	}

	text := requestText(req)
	prepared := prepareText(text)

	if len(prepared.text) > maxBytes {
		return xpost.ValidationError{
			Provider: providerName,
			Reason:   fmt.Sprintf("message too long: %d bytes (max %d)", len(prepared.text), maxBytes),
		}
	}

	count := uniseg.GraphemeClusterCount(prepared.text)
	if count > maxGraphemes {
		return xpost.ValidationError{
			Provider: providerName,
			Reason:   fmt.Sprintf("message too long: %d graphemes (max %d)", count, maxGraphemes),
		}
	}
	if req.ReplyTo != nil {
		if _, err := strongRef(req.ReplyTo); err != nil {
			return err
		}

		root := req.RootReplyTo
		if root == nil {
			root = req.ReplyTo
		}
		if _, err := strongRef(root); err != nil {
			return err
		}
	}

	return nil
}

// Post creates a new Bluesky post with an optional image embed.
func (c *Client) Post(ctx context.Context, req xpost.Request) error {
	_, err := c.PostResult(ctx, req)
	return err
}

// PostResult creates a new Bluesky post and returns its remote identity.
func (c *Client) PostResult(ctx context.Context, req xpost.Request) (xpost.Result, error) {
	// Build the text, appending link if provided
	prepared := prepareText(requestText(req))

	post := &bsky.FeedPost{
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Text:      prepared.text,
		Facets:    prepared.facets,
	}

	if req.ReplyTo != nil {
		parent, err := strongRef(req.ReplyTo)
		if err != nil {
			return xpost.Result{}, err
		}
		root := req.RootReplyTo
		if root == nil {
			root = req.ReplyTo
		}
		rootRef, err := strongRef(root)
		if err != nil {
			return xpost.Result{}, err
		}
		post.Reply = &bsky.FeedPost_ReplyRef{
			Parent: parent,
			Root:   rootRef,
		}
	}

	if len(req.Attachments) > 0 {
		images := make([]*bsky.EmbedImages_Image, 0, len(req.Attachments))
		for _, attachment := range req.Attachments {
			blob, err := c.uploadImage(ctx, attachment.Path)
			if err != nil {
				return xpost.Result{}, err
			}
			images = append(images, &bsky.EmbedImages_Image{
				Alt:   attachment.Alt,
				Image: blob,
			})
		}
		post.Embed = &bsky.FeedPost_Embed{
			EmbedImages: &bsky.EmbedImages{
				Images: images,
			},
		}
	}

	response, err := atproto.RepoCreateRecord(ctx, c.client, &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       c.client.Auth.Did,
		Record: &util.LexiconTypeDecoder{
			Val: post,
		},
	})
	if err != nil {
		return xpost.Result{}, fmt.Errorf("create record: %w", err)
	}
	if response == nil || response.Uri == "" || response.Cid == "" {
		return xpost.Result{}, errors.New("create record: empty response")
	}

	return xpost.Result{
		RemoteID:  response.Uri,
		RemoteCID: response.Cid,
		URL:       postURL(response.Uri, c.client.Auth.Handle),
	}, nil
}

func strongRef(ref *xpost.Reference) (*atproto.RepoStrongRef, error) {
	if ref == nil || strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.CID) == "" {
		return nil, xpost.ValidationError{
			Provider: providerName,
			Reason:   "reply reference requires an id and cid",
		}
	}
	return &atproto.RepoStrongRef{Uri: ref.ID, Cid: ref.CID}, nil
}

func postURL(uri, handle string) string {
	parts := strings.Split(strings.TrimPrefix(uri, "at://"), "/")
	if len(parts) != 3 || parts[1] != "app.bsky.feed.post" || handle == "" {
		return ""
	}
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, parts[2])
}

func (c *Client) uploadImage(ctx context.Context, path string) (*util.LexBlob, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, xpost.ValidationError{Provider: providerName, Reason: fmt.Sprintf("image %q not found", path)}
		}
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	resp, err := atproto.RepoUploadBlob(ctx, c.client, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("upload blob: %w", err)
	}

	if resp.Blob == nil {
		return nil, fmt.Errorf("upload blob: empty response")
	}

	return resp.Blob, nil
}

// ProviderConfig merges defaults with environment-defined values.
type ProviderConfig struct {
	Handle      string
	AppPassword string
	PDSURL      string
}

func loadConfig(base Config) (ProviderConfig, error) {
	cfg := ProviderConfig{
		Handle:      strings.TrimSpace(base.Handle),
		AppPassword: strings.TrimSpace(base.AppPassword),
		PDSURL:      strings.TrimSpace(base.PDSURL),
	}
	if cfg.PDSURL == "" {
		cfg.PDSURL = "https://bsky.social"
	}

	var missing []string
	if cfg.Handle == "" {
		missing = append(missing, envHandle)
	}
	if cfg.AppPassword == "" {
		missing = append(missing, envAppPassword)
	}
	if cfg.PDSURL == "" {
		missing = append(missing, envPDSURL)
	}

	if len(missing) > 0 {
		return ProviderConfig{}, xpost.MissingEnvError{Provider: providerName, Variables: missing}
	}

	return cfg, nil
}
