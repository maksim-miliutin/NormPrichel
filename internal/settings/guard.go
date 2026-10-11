package settings

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var ErrKey = errors.New("settings: no key could be made for the page")

const policy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func NewKey(random io.Reader) (string, error) {
	var raw [32]byte
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", fmt.Errorf("%w: %w", ErrKey, err)
	}

	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// Guard lets through only requests that carry the key and name the address the page was
// opened at: a site that points its own name at 127.0.0.1 still sends that name as the Host.
func Guard(key, host string, page http.Handler) http.Handler {
	inner := http.StripPrefix("/"+key, page)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segment, _, found := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if r.Host != host || !found || subtle.ConstantTimeCompare([]byte(segment), []byte(key)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		headers := w.Header()
		headers.Set("Cache-Control", "no-store")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("Content-Security-Policy", policy)

		inner.ServeHTTP(w, r)
	})
}
