package egress

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fil-forge/automobile"
	"github.com/ipfs/go-cid"
)

// NewHandler creates an [http.Handler] that serves the journal's sealed
// batches to the egress tracking service. It expects the URL path to end with
// the batch CID, e.g. /egress/<batch-cid>.
func NewHandler(journal *Journal) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimSuffix(r.URL.Path, "/")
		batch, err := cid.Parse(path[strings.LastIndex(path, "/")+1:])
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid batch CID: %v", err), http.StatusBadRequest)
			return
		}
		f, err := journal.Open(batch)
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "batch not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Errorw("opening batch", "batch", batch, "error", err)
			http.Error(w, "failed to open batch", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			log.Errorw("opening batch", "batch", batch, "error", err)
			http.Error(w, "failed to open batch", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", automobile.ContentType)
		// A batch never changes, so ServeContent's range and conditional
		// handling is safe, and the file's time is as good as any.
		http.ServeContent(w, r, "", info.ModTime(), f)
	})
}
