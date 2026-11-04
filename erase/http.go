package erase

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Handler serves DELETE /internal/accounts/{account_id}/data with
// "Authorization: Bearer <key>". An empty key serves 503, never an open door.
func Handler(key string, run func(r *http.Request, accountID string) (int, error)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /internal/accounts/{account_id}/data", func(w http.ResponseWriter, r *http.Request) {
		if key == "" {
			http.Error(w, `{"error":"erasure not configured"}`, http.StatusServiceUnavailable)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") ||
			subtle.ConstantTimeCompare([]byte(got), []byte(key)) != 1 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		id := r.PathValue("account_id")
		if !uuidRe.MatchString(id) {
			http.Error(w, `{"error":"account_id must be a UUID"}`, http.StatusBadRequest)
			return
		}
		rows, err := run(r, strings.ToLower(id))
		if err != nil {
			http.Error(w, `{"error":"erase failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"service": Service, "rows": rows})
	})
	return mux
}
