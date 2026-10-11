//go:build integration

package settings_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

func TestThePageListensOnThisComputerAloneAndAnswersOnlyWithTheKey(t *testing.T) {
	listener, err := settings.Listen()
	if err != nil {
		t.Fatal(err)
	}
	host := listener.Addr().String()
	if address, _, _ := net.SplitHostPort(host); !net.ParseIP(address).IsLoopback() {
		t.Errorf("the page listens on %s, want the loopback address", host)
	}

	state := live.New(listed(), nowhere{}, func(error) {})
	defer state.Close()
	page := settings.Page{State: state, Taken: (&site{}).held, Language: "en"}
	server := &http.Server{Handler: settings.Guard(key, host, page.Handler())}
	go server.Serve(listener)
	defer server.Close()

	cases := map[string]struct {
		path, hostHeader string
		want             int
	}{
		"with the key":                  {path: "/" + key + "/api/state", want: http.StatusOK},
		"without the key":               {path: "/api/state", want: http.StatusForbidden},
		"with the key under a new name": {path: "/" + key + "/api/state", hostHeader: "evil.example", want: http.StatusForbidden},
	}
	for name, c := range cases {
		request, err := http.NewRequest(http.MethodGet, "http://"+host+c.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if c.hostHeader != "" {
			request.Host = c.hostHeader
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		response.Body.Close()
		if response.StatusCode != c.want {
			t.Errorf("%s: status %d, want %d", name, response.StatusCode, c.want)
		}
	}
}
