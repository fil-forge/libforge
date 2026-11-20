package diddoc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fil-forge/libforge/diddoc"
	"github.com/fil-forge/libforge/identity"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/did/web"
	"github.com/stretchr/testify/require"
)

func newDocument(t *testing.T, serviceDID string) did.Document {
	t.Helper()
	id, err := identity.New("", serviceDID)
	require.NoError(t, err)
	doc, err := id.DIDDocument()
	require.NoError(t, err)
	return doc
}

func serve(t *testing.T, h http.Handler, method string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, web.WellKnownDIDPath, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler(t *testing.T) {
	doc := newDocument(t, "did:web:example.com")
	modified := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	const modifiedHeader = "Thu, 01 Jan 2026 12:00:00 GMT"

	newHandler := func(t *testing.T, options ...diddoc.Option) http.Handler {
		t.Helper()
		h, err := diddoc.NewHandler(doc, append([]diddoc.Option{diddoc.WithLastModified(modified)}, options...)...)
		require.NoError(t, err)
		return h
	}

	t.Run("serves the document", func(t *testing.T) {
		rec := serve(t, newHandler(t), http.MethodGet, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		require.Equal(t, "max-age=300", rec.Header().Get("Cache-Control"))
		require.Equal(t, modifiedHeader, rec.Header().Get("Last-Modified"))
		require.Regexp(t, `^"[0-9a-f]{32}"$`, rec.Header().Get("ETag"))

		want, err := json.Marshal(doc)
		require.NoError(t, err)
		require.JSONEq(t, string(want), rec.Body.String())
	})

	t.Run("answers HEAD without a body", func(t *testing.T) {
		rec := serve(t, newHandler(t), http.MethodHead, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Body.Bytes())
		require.NotEmpty(t, rec.Header().Get("ETag"))
		require.NotEmpty(t, rec.Header().Get("Content-Length"))
	})

	t.Run("refuses other methods", func(t *testing.T) {
		rec := serve(t, newHandler(t), http.MethodPost, nil)
		require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		require.Equal(t, "GET, HEAD", rec.Header().Get("Allow"))
	})

	t.Run("TTL", func(t *testing.T) {
		tests := []struct {
			name string
			ttl  time.Duration
			want string
		}{
			{"minutes", 10 * time.Minute, "max-age=600"},
			{"fraction of a second dropped", 1500 * time.Millisecond, "max-age=1"},
			{"zero", 0, "no-cache"},
			{"under a second", 500 * time.Millisecond, "no-cache"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rec := serve(t, newHandler(t, diddoc.WithTTL(tt.ttl)), http.MethodGet, nil)
				require.Equal(t, tt.want, rec.Header().Get("Cache-Control"))
			})
		}
	})

	t.Run("conditional requests", func(t *testing.T) {
		h := newHandler(t)
		etag := serve(t, h, http.MethodGet, nil).Header().Get("ETag")

		tests := []struct {
			name   string
			header http.Header
			want   int
		}{
			{"matching If-None-Match", http.Header{"If-None-Match": {etag}}, http.StatusNotModified},
			{"weak matching If-None-Match", http.Header{"If-None-Match": {"W/" + etag}}, http.StatusNotModified},
			{"If-None-Match list", http.Header{"If-None-Match": {`"other", ` + etag}}, http.StatusNotModified},
			{"If-None-Match any", http.Header{"If-None-Match": {"*"}}, http.StatusNotModified},
			{"other If-None-Match", http.Header{"If-None-Match": {`"other"`}}, http.StatusOK},
			{"If-Modified-Since at the modification time", http.Header{"If-Modified-Since": {modifiedHeader}}, http.StatusNotModified},
			{"If-Modified-Since after it", http.Header{"If-Modified-Since": {"Fri, 02 Jan 2026 12:00:00 GMT"}}, http.StatusNotModified},
			{"If-Modified-Since before it", http.Header{"If-Modified-Since": {"Wed, 31 Dec 2025 12:00:00 GMT"}}, http.StatusOK},
			{
				"If-None-Match takes precedence over If-Modified-Since",
				http.Header{"If-None-Match": {`"other"`}, "If-Modified-Since": {modifiedHeader}},
				http.StatusOK,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rec := serve(t, h, http.MethodGet, tt.header)
				require.Equal(t, tt.want, rec.Code)
				require.Equal(t, etag, rec.Header().Get("ETag"))
				require.Equal(t, "max-age=300", rec.Header().Get("Cache-Control"))
				if tt.want == http.StatusNotModified {
					require.Empty(t, rec.Body.Bytes())
				} else {
					require.NotEmpty(t, rec.Body.Bytes())
				}
			})
		}
	})

	t.Run("ETag follows the document", func(t *testing.T) {
		etag := func(doc did.Document) string {
			h, err := diddoc.NewHandler(doc)
			require.NoError(t, err)
			return serve(t, h, http.MethodGet, nil).Header().Get("ETag")
		}
		require.Equal(t, etag(doc), etag(doc))
		require.NotEqual(t, etag(doc), etag(newDocument(t, "did:web:example.com")))
	})

	t.Run("Last-Modified defaults to the time the handler was created", func(t *testing.T) {
		before := time.Now().Truncate(time.Second)
		h, err := diddoc.NewHandler(doc)
		require.NoError(t, err)
		got, err := http.ParseTime(serve(t, h, http.MethodGet, nil).Header().Get("Last-Modified"))
		require.NoError(t, err)
		require.False(t, got.Before(before))
		require.False(t, got.After(time.Now()))
	})

	t.Run("refuses invalid options", func(t *testing.T) {
		_, err := diddoc.NewHandler(doc, diddoc.WithTTL(-time.Second))
		require.ErrorContains(t, err, "TTL cannot be negative")
		_, err = diddoc.NewHandler(doc, diddoc.WithLastModified(time.Time{}))
		require.ErrorContains(t, err, "last modified time cannot be zero")
	})
}
