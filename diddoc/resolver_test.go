package diddoc_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fil-forge/libforge/diddoc"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/did/web"
	"github.com/stretchr/testify/require"
)

// A ucantone did:web resolver pointed at the handler caches the document and
// revalidates it with a conditional request.
func TestHandlerWithResolver(t *testing.T) {
	var statuses []int
	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		statuses = append(statuses, rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
	}))
	defer srv.Close()

	// A did:web names a host, never an IP address, so the test server is
	// addressed as localhost.
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)
	serviceDID := did.MustParse("did:web:localhost%3A" + port)
	doc := newDocument(t, serviceDID.String())
	handler, err = diddoc.NewHandler(doc, diddoc.WithTTL(0))
	require.NoError(t, err)

	resolver, err := web.NewResolver(web.WithInsecure(true))
	require.NoError(t, err)

	for range 3 {
		got, err := resolver.Resolve(t.Context(), serviceDID)
		require.NoError(t, err)
		require.Equal(t, doc.ID, got.ID)
		require.Equal(t, doc.VerificationMethods, got.VerificationMethods)
	}
	require.Equal(t, []int{http.StatusOK, http.StatusNotModified, http.StatusNotModified}, statuses)
}
