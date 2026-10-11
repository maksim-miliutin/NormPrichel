package settings_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

func pageIn(t *testing.T, language string) http.Handler {
	t.Helper()

	state := live.New(listed(), nowhere{}, func(error) {})
	t.Cleanup(state.Close)

	return settings.Page{State: state, Language: language, Taken: (&site{}).held}.Handler()
}

func get(t *testing.T, page http.Handler, path string) (string, string) {
	t.Helper()

	st := &site{page: page}
	w := st.call(t, http.MethodGet, path, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d", path, w.Code)
	}

	return w.Body.String(), w.Header().Get("Content-Type")
}

type worded struct {
	Settings profiles.Settings `json:"settings"`
	Shapes   []string          `json:"shapes"`
	Text     map[string]string `json:"text"`
	Arrows   []string          `json:"arrows"`
}

func wordsIn(t *testing.T, language string) worded {
	t.Helper()

	body, _ := get(t, pageIn(t, language), "/api/state")
	var got worded
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}

	return got
}

func TestThePageComesWithItsScriptAndStyleTypedWhateverWindowsSays(t *testing.T) {
	page := pageIn(t, "ru")

	types := map[string]string{
		"/":          "text/html; charset=utf-8",
		"/app.js":    "text/javascript; charset=utf-8",
		"/style.css": "text/css; charset=utf-8",
	}
	for path, want := range types {
		if _, got := get(t, page, path); got != want {
			t.Errorf("%s comes as %q, want %q", path, got, want)
		}
	}

	st := &site{page: page}
	for _, path := range []string{"/index.html", "/page/app.js", "/missing.js"} {
		if w := st.call(t, http.MethodGet, path, ""); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, w.Code)
		}
	}
}

func shown(t *testing.T, page http.Handler, words worded) []string {
	t.Helper()

	html, _ := get(t, page, "/")
	script, _ := get(t, page, "/app.js")

	var keys []string
	for _, pattern := range []string{`data-text="([^"]+)"`, `\bt\("([^"]+)"\)`, `key: "([^"]+)"`} {
		for _, match := range regexp.MustCompile(pattern).FindAllStringSubmatch(html+script, -1) {
			keys = append(keys, match[1])
		}
	}
	if len(keys) < 40 {
		t.Fatalf("found only %d words on the page; the patterns no longer match it", len(keys))
	}

	for _, shape := range words.Shapes {
		keys = append(keys, "shape."+shape)
	}
	fields, _ := json.Marshal(words.Settings.Hotkeys)
	var hotkeys map[string]string
	json.Unmarshal(fields, &hotkeys)
	for field := range hotkeys {
		keys = append(keys, "hotkey."+field)
	}

	return keys
}

func TestEveryWordThePageUsesIsThereInBothLanguages(t *testing.T) {
	page := pageIn(t, "ru")

	for _, language := range []string{"ru", "en"} {
		words := wordsIn(t, language)
		for _, key := range shown(t, page, words) {
			if strings.TrimSpace(words.Text[key]) == "" {
				t.Errorf("%s: no words for %q", language, key)
			}
		}
	}
}

// Refusals are worded after the program's reasons, which a test inside the package holds.
func TestThePageShowsEveryWordItIsSent(t *testing.T) {
	words := wordsIn(t, "en")
	used := map[string]bool{}
	for _, key := range shown(t, pageIn(t, "en"), words) {
		used[key] = true
	}

	for key := range words.Text {
		if !used[key] && !strings.HasPrefix(key, "error.") {
			t.Errorf("%q is worded, but nothing on the page shows it", key)
		}
	}
}

// Square is the design: the browser rounds its own controls, so the style takes that off every
// element, and nothing of its own puts it back.
func TestNothingOnThePageIsRounded(t *testing.T) {
	style, _ := get(t, pageIn(t, "ru"), "/style.css")

	if !regexp.MustCompile(`(?m)^\*\s*\{[^}]*\bborder-radius: 0;`).MatchString(style) {
		t.Error("the style leaves the browser's rounding on")
	}
	for _, radius := range regexp.MustCompile(`border-radius:\s*([^;]*);`).FindAllStringSubmatch(style, -1) {
		if radius[1] != "0" {
			t.Errorf("a corner is rounded by %s", radius[1])
		}
	}
}

func TestBothLanguagesNameTheSameThings(t *testing.T) {
	russian, english := wordsIn(t, "ru").Text, wordsIn(t, "en").Text

	for key := range russian {
		if _, ok := english[key]; !ok {
			t.Errorf("%q is worded in Russian only", key)
		}
	}
	for key := range english {
		if _, ok := russian[key]; !ok {
			t.Errorf("%q is worded in English only", key)
		}
	}
	if russian["games"] == english["games"] {
		t.Errorf("both languages say %q", russian["games"])
	}
}

func TestAnyOtherLanguageGetsEnglish(t *testing.T) {
	if got := wordsIn(t, "de").Text["games"]; got != "Games" {
		t.Errorf("got %q, want the English word", got)
	}
}

func TestTheNudgeFieldsAreTheOnesHeldWithArrows(t *testing.T) {
	if got := wordsIn(t, "en").Arrows; strings.Join(got, ",") != "nudge,nudgeFar" {
		t.Errorf("got %v, want nudge and nudgeFar", got)
	}
}

// The page holds the key, so nothing on it may turn text into code: no inline scripts or
// handlers for the content policy to refuse, and no HTML built from strings.
func TestThePageRunsNoCodeMadeOfText(t *testing.T) {
	page := pageIn(t, "ru")
	html, _ := get(t, page, "/")
	script, _ := get(t, page, "/app.js")

	for _, tag := range regexp.MustCompile(`<script[^>]*>([^<]*)</script>`).FindAllStringSubmatch(html, -1) {
		if !strings.Contains(tag[0], " src=") || strings.TrimSpace(tag[1]) != "" {
			t.Errorf("an inline script: %s", tag[0])
		}
	}
	if handler := regexp.MustCompile(`\son[a-z]+=|\sstyle=`).FindString(html); handler != "" {
		t.Errorf("an inline handler or style: %q", handler)
	}
	for _, sink := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function"} {
		if strings.Contains(script, sink) {
			t.Errorf("the script uses %s", sink)
		}
	}
}
