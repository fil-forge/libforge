// Package diddoc serves a DID document over HTTP, for services that publish
// their did:web identity at /.well-known/did.json.
package diddoc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fil-forge/ucantone/did"
)

// DefaultTTL is how long clients may cache the document when [WithTTL] is not
// given.
const DefaultTTL = 5 * time.Minute

type config struct {
	ttl          time.Duration
	lastModified time.Time
}

type Option func(*config) error

// WithTTL sets how long clients may cache the document before asking for it
// again, sent as the Cache-Control max-age. The default is [DefaultTTL]. It
// is how long a key removed from the document can go on being trusted by
// those who already fetched it, so keep it short enough for a key rotation.
// Whole seconds are sent; a fraction is dropped. Zero tells clients to
// revalidate on every use (Cache-Control: no-cache).
func WithTTL(ttl time.Duration) Option {
	return func(c *config) error {
		if ttl < 0 {
			return fmt.Errorf("TTL cannot be negative")
		}
		c.ttl = ttl
		return nil
	}
}

// WithLastModified sets the time the document last changed, sent as the
// Last-Modified header. The default is the time the handler was created.
func WithLastModified(t time.Time) Option {
	return func(c *config) error {
		if t.IsZero() {
			return fmt.Errorf("last modified time cannot be zero")
		}
		c.lastModified = t
		return nil
	}
}

// NewHandler creates an [http.Handler] that serves doc as JSON in answer to
// GET and HEAD requests, whatever the request path. Mount it at
// /.well-known/did.json (ucantone's web.WellKnownDIDPath) for a did:web DID
// with no path.
//
// Responses carry a Cache-Control max-age (see [WithTTL]), an ETag derived
// from the document's content and a Last-Modified time, and the handler
// answers If-None-Match and If-Modified-Since requests with 304 Not Modified
// when the client's copy is current. The ETag is the same in every process
// serving the same document, so revalidation works across restarts and
// replicas.
//
// The document is encoded once, here. To serve a changed document, create a
// new handler.
func NewHandler(doc did.Document, options ...Option) (http.Handler, error) {
	cfg := config{ttl: DefaultTTL, lastModified: time.Now()}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encoding DID document: %w", err)
	}
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:16]) + `"`

	cacheControl := "no-cache"
	if seconds := int64(cfg.ttl / time.Second); seconds > 0 {
		cacheControl = "max-age=" + strconv.FormatInt(seconds, 10)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h := w.Header()
		h.Set("Content-Type", "application/json")
		h.Set("Cache-Control", cacheControl)
		h.Set("ETag", etag)
		// ServeContent sets Last-Modified and answers conditional requests.
		http.ServeContent(w, r, "", cfg.lastModified, bytes.NewReader(body))
	}), nil
}
