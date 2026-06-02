package prober_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"cactus/config"
	"cactus/notifier"
	"cactus/prober"
)

// probe runs Run in a goroutine and returns a stop func.
func probe(cfg *config.ProbeConfig, n notifier.Notifier) (stop func()) {
	ch := make(chan struct{})
	go prober.Run(cfg, n, ch)
	return func() { close(ch) }
}

func probeConfig(url string) *config.ProbeConfig {
	return &config.ProbeConfig{
		Name:           "test",
		URL:            url,
		Interval:       config.Duration{Duration: 20 * time.Millisecond},
		ExpectedStatus: 200,
	}
}

// --- mock notifier ---

type mockNotifier struct {
	ch chan mockAlert
}

type mockAlert struct {
	up     bool
	errMsg string
}

func newMock() *mockNotifier {
	return &mockNotifier{ch: make(chan mockAlert, 10)}
}

func (m *mockNotifier) Alert(_, _ string, up bool, errMsg string) {
	m.ch <- mockAlert{up, errMsg}
}

func (m *mockNotifier) recv(t *testing.T, timeout time.Duration) mockAlert {
	t.Helper()
	select {
	case a := <-m.ch:
		return a
	case <-time.After(timeout):
		t.Fatal("timed out waiting for alert")
		return mockAlert{}
	}
}

func (m *mockNotifier) expectNone(t *testing.T, window time.Duration) {
	t.Helper()
	select {
	case a := <-m.ch:
		t.Errorf("unexpected alert: up=%v errMsg=%q", a.up, a.errMsg)
	case <-time.After(window):
	}
}

// --- tests ---

func TestRun_AlertsOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	mock := newMock()
	stop := probe(probeConfig(srv.URL), mock)
	defer stop()

	alert := mock.recv(t, time.Second)
	if alert.up {
		t.Error("want down alert, got up")
	}
}

func TestRun_NoAlertOnSteadyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mock := newMock()
	stop := probe(probeConfig(srv.URL), mock)
	defer stop()

	mock.expectNone(t, 100*time.Millisecond)
}

func TestRun_AlertOnlyOnceForRepeatedFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	mock := newMock()
	stop := probe(probeConfig(srv.URL), mock)
	defer stop()

	// First alert expected.
	mock.recv(t, time.Second)
	// No second alert for sustained failure.
	mock.expectNone(t, 100*time.Millisecond)
}

func TestRun_RecoveryAlert(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	mock := newMock()
	stop := probe(probeConfig(srv.URL), mock)
	defer stop()

	down := mock.recv(t, time.Second)
	if down.up {
		t.Error("first alert: want down, got up")
	}

	up := mock.recv(t, time.Second)
	if !up.up {
		t.Error("second alert: want up (recovery), got down")
	}
}

func TestRun_BasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := probeConfig(srv.URL)
	cfg.Auth = &config.AuthConfig{Username: "alice", Password: "secret"}

	mock := newMock()
	stop := probe(cfg, mock)
	defer stop()

	mock.expectNone(t, 80*time.Millisecond)

	// "alice:secret" base64-encoded is "YWxpY2U6c2VjcmV0"
	want := "Basic YWxpY2U6c2VjcmV0"
	if gotAuth != want {
		t.Errorf("Authorization header: got %q, want %q", gotAuth, want)
	}
}

func TestRun_FollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirect.Close()

	mock := newMock()
	stop := probe(probeConfig(redirect.URL), mock)
	defer stop()

	mock.expectNone(t, 100*time.Millisecond)
}

func TestRun_CustomHeaders(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := probeConfig(srv.URL)
	cfg.Headers = map[string]string{"X-Token": "abc123"}

	mock := newMock()
	stop := probe(cfg, mock)
	defer stop()

	mock.expectNone(t, 80*time.Millisecond)

	if gotHeader != "abc123" {
		t.Errorf("X-Token: got %q, want %q", gotHeader, "abc123")
	}
}
