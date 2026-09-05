package xpost

import "context"

const (
	BridgeOperationPublish  = "publish"
	BridgeOperationValidate = "validate"
	BridgeStatusPublished   = "published"
	BridgeStatusValidated   = "validated"
	BridgeStatusFailed      = "failed"
	BridgeStatusRejected    = "rejected"
)

// Attachment describes one local image and its alternative text.
type Attachment struct {
	Path string `json:"path"`
	Alt  string `json:"alt,omitempty"`
}

// Reference identifies a previously published post.
type Reference struct {
	ID  string `json:"id"`
	CID string `json:"cid,omitempty"`
}

// Request defines the message payload shared across all providers.
type Request struct {
	Message     string
	Link        string // Optional URL to append to message with proper formatting
	Attachments []Attachment
	ReplyTo     *Reference
	RootReplyTo *Reference
}

// Result contains the remote identity returned after publication.
type Result struct {
	RemoteID  string
	RemoteCID string
	URL       string
}

// Poster abstracts a social network that can publish content.
type Poster interface {
	Name() string
	// Validate checks if the request meets platform constraints (character limits, etc.)
	// without posting. Returns nil if valid.
	Validate(req Request) error
	Post(ctx context.Context, req Request) error
}

// ResultPoster publishes content and returns the remote identity.
type ResultPoster interface {
	PostResult(ctx context.Context, req Request) (Result, error)
}

// BridgeRequest is the JSON input accepted by the bridge command.
type BridgeRequest struct {
	Operation   string       `json:"operation"`
	Target      string       `json:"target"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
	ReplyTo     *Reference   `json:"reply_to,omitempty"`
	RootReplyTo *Reference   `json:"root_reply_to,omitempty"`
}

// BridgeResponse is the JSON output emitted by the bridge command.
type BridgeResponse struct {
	Status    string `json:"status"`
	RemoteID  string `json:"remote_id,omitempty"`
	RemoteCID string `json:"remote_cid,omitempty"`
	URL       string `json:"url,omitempty"`
	Error     string `json:"error,omitempty"`
	ErrorKind string `json:"error_kind,omitempty"`
}
