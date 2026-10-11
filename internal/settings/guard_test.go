package settings_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

const (
	key  = "hPbn2gAeQdW8mHq7vWkXQ1sXo3Yh0d7mU5uZ0kIY4Zs"
	host = "127.0.0.1:51234"
)

// inner stands for the page behind the guard and remembers the paths that reached it.
type inner struct {
	paths []string
}

func (i *inner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.paths = append(i.paths, r.URL.Path)
}

func ask(handler http.Handler, method, hostname, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+hostname+path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	return w
}

func TestARequestWithTheKeyFromThisComputerGetsThrough(t *testing.T) {
	page := &inner{}
	guard := settings.Guard(key, host, page)

	got := ask(guard, http.MethodGet, host, "/"+key+"/api/state")

	if got.Code != http.StatusOK || len(page.paths) != 1 || page.paths[0] != "/api/state" {
		t.Errorf("status %d, the page saw %v; want 200 and /api/state without the key", got.Code, page.paths)
	}
}

func TestARequestWithoutTheKeyIsTurnedAway(t *testing.T) {
	paths := []string{
		"/",
		"/api/state",
		"/" + key,
		"/" + key + "x/api/state",
		"/x" + key + "/api/state",
		"/" + key[:len(key)-1] + "/api/state",
		"/" + strings.ToLower(key) + "/api/state",
	}

	for _, path := range paths {
		page := &inner{}
		got := ask(settings.Guard(key, host, page), http.MethodGet, host, path)

		if got.Code != http.StatusForbidden || len(page.paths) != 0 {
			t.Errorf("%s: status %d, the page saw %v; want 403 and nothing let through", path, got.Code, page.paths)
		}
	}
}

func TestAnotherHostIsTurnedAwayEvenWithTheKey(t *testing.T) {
	hosts := []string{"evil.example:51234", "localhost:51234", "127.0.0.1:51235", "127.0.0.1", "[::1]:51234"}

	for _, other := range hosts {
		page := &inner{}
		got := ask(settings.Guard(key, host, page), http.MethodGet, other, "/"+key+"/api/state")

		if got.Code != http.StatusForbidden || len(page.paths) != 0 {
			t.Errorf("host %s: status %d, the page saw %v; want 403", other, got.Code, page.paths)
		}
	}
}

func TestARequestFromAnotherSiteIsTurnedAwayEvenWithTheKey(t *testing.T) {
	sites := map[string]int{
		"":            http.StatusOK,
		"none":        http.StatusOK,
		"same-origin": http.StatusOK,
		"same-site":   http.StatusForbidden,
		"cross-site":  http.StatusForbidden,
	}

	for site, want := range sites {
		page := &inner{}
		r := httptest.NewRequest(http.MethodPost, "http://"+host+"/"+key+"/api/games", nil)
		if site != "" {
			r.Header.Set("Sec-Fetch-Site", site)
		}
		w := httptest.NewRecorder()
		settings.Guard(key, host, page).ServeHTTP(w, r)

		if w.Code != want || (want == http.StatusForbidden) != (len(page.paths) == 0) {
			t.Errorf("Sec-Fetch-Site %q: status %d with %d let through, want %d", site, w.Code, len(page.paths), want)
		}
	}
}

func TestAnAnswerIsNotCachedReferredOnOrFramed(t *testing.T) {
	got := ask(settings.Guard(key, host, &inner{}), http.MethodGet, host, "/"+key+"/")

	headers := map[string]string{
		"Cache-Control":          "no-store",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
	}
	for name, want := range headers {
		if value := got.Header().Get(name); value != want {
			t.Errorf("%s is %q, want %q", name, value, want)
		}
	}

	policy := got.Header().Get("Content-Security-Policy")
	for _, rule := range []string{"default-src 'none'", "script-src 'self'", "connect-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(policy, rule) {
			t.Errorf("the content policy %q lacks %q", policy, rule)
		}
	}
}

func TestAKeyIs32RandomBytesWrittenForAPath(t *testing.T) {
	first, err := settings.NewKey(bytes.NewReader(bytes.Repeat([]byte{0xFB}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	second, err := settings.NewKey(bytes.NewReader(bytes.Repeat([]byte{0x07}, 32)))
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != 43 || strings.ContainsAny(first, "+/=") || first == second {
		t.Errorf("keys %q and %q, want 43 characters each, safe in a path, and different", first, second)
	}
}

func TestAShortRandomSourceMakesNoKey(t *testing.T) {
	if _, err := settings.NewKey(bytes.NewReader(make([]byte, 31))); !errors.Is(err, settings.ErrKey) {
		t.Errorf("got %v, want ErrKey", err)
	}
}
