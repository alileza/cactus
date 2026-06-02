package notifier_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"cactus/notifier"
)

// --- multi ---

type recorder struct {
	calls []string
}

func (r *recorder) Alert(name, _ string, up bool, _ string) {
	state := "down"
	if up {
		state = "up"
	}
	r.calls = append(r.calls, name+":"+state)
}

func TestMulti_FanOut(t *testing.T) {
	a, b := &recorder{}, &recorder{}
	m := notifier.Multi(a, b)
	m.Alert("svc", "http://x", false, "err")

	for _, r := range []*recorder{a, b} {
		if len(r.calls) != 1 || r.calls[0] != "svc:down" {
			t.Errorf("got calls %v, want [svc:down]", r.calls)
		}
	}
}

func TestMulti_Empty(t *testing.T) {
	// Must not panic.
	notifier.Multi().Alert("svc", "http://x", false, "err")
}

// --- telegram ---

func TestTelegram_Alert_Down(t *testing.T) {
	var body url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		body = r.Form
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tg := notifier.NewTelegramWithEndpoint(notifier.TelegramConfig{
		BotToken: "tok",
		ChatID:   "42",
	}, srv.URL+"/bot%s/sendMessage")

	tg.Alert("api", "https://example.com", false, "timeout")

	if body.Get("chat_id") != "42" {
		t.Errorf("chat_id: got %q, want %q", body.Get("chat_id"), "42")
	}
	text := body.Get("text")
	if !strings.Contains(text, "[FIRING]") {
		t.Errorf("text missing [FIRING]: %q", text)
	}
	if !strings.Contains(text, "timeout") {
		t.Errorf("text missing error message: %q", text)
	}
}

func TestTelegram_Alert_Up(t *testing.T) {
	var text string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		text = r.FormValue("text")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tg := notifier.NewTelegramWithEndpoint(notifier.TelegramConfig{
		BotToken: "tok",
		ChatID:   "42",
	}, srv.URL+"/bot%s/sendMessage")

	tg.Alert("api", "https://example.com", true, "")

	if !strings.Contains(text, "[RESOLVED]") {
		t.Errorf("text missing [RESOLVED]: %q", text)
	}
}

func TestTelegram_Alert_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tg := notifier.NewTelegramWithEndpoint(notifier.TelegramConfig{
		BotToken: "tok",
		ChatID:   "42",
	}, srv.URL+"/bot%s/sendMessage")

	// Must not panic; error is logged internally.
	tg.Alert("api", "https://example.com", false, "err")
}
