package settings

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

var ErrKey = errors.New("settings: no key could be made for the page")

const policy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// Listen takes a free port on the loopback address alone: nothing outside this computer can reach it.
func Listen() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func NewKey(random io.Reader) (string, error) {
	var raw [32]byte
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", fmt.Errorf("%w: %w", ErrKey, err)
	}

	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// Guard lets through only the page's own requests: with the key, to the address it was opened
// at (a site can borrow 127.0.0.1 by DNS, not the Host) and, by the browser's word, not from a site.
func Guard(key, host string, page http.Handler) http.Handler {
	inner := http.StripPrefix("/"+key, page)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segment, _, found := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		keyed := found && subtle.ConstantTimeCompare([]byte(segment), []byte(key)) == 1
		site := r.Header.Get("Sec-Fetch-Site")
		if r.Host != host || !keyed || site != "" && site != "same-origin" && site != "none" {
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
