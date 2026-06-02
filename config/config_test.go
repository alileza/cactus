package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cactus/config"
)

func TestLoad(t *testing.T) {
	yaml := `
probes:
  - name: api
    url: https://example.com/health
    method: GET
    interval: 45s
    expected_status: 204
    auth:
      username: user
      password: pass
    headers:
      X-Custom: value
receivers:
  telegram:
    bot_token: tok
    chat_id: "123"
`
	path := writeTemp(t, yaml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Probes) != 1 {
		t.Fatalf("want 1 probe, got %d", len(cfg.Probes))
	}
	p := cfg.Probes[0]

	check := func(field, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %q, want %q", field, got, want)
		}
	}
	check("Name", p.Name, "api")
	check("URL", p.URL, "https://example.com/health")
	check("Method", p.Method, "GET")
	check("Headers[X-Custom]", p.Headers["X-Custom"], "value")

	if p.ExpectedStatus != 204 {
		t.Errorf("ExpectedStatus: got %d, want 204", p.ExpectedStatus)
	}
	if p.Interval.Duration != 45*time.Second {
		t.Errorf("Interval: got %v, want 45s", p.Interval.Duration)
	}
	if p.Auth == nil {
		t.Fatal("Auth: want non-nil")
	}
	check("Auth.Username", p.Auth.Username, "user")
	check("Auth.Password", p.Auth.Password, "pass")

	if cfg.Receivers.Telegram == nil {
		t.Fatal("Receivers.Telegram: want non-nil")
	}
	check("BotToken", cfg.Receivers.Telegram.BotToken, "tok")
	check("ChatID", cfg.Receivers.Telegram.ChatID, "123")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTemp(t, "probes: [\n  invalid")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("want error for invalid YAML, got nil")
	}
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	cases := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"30s", 30 * time.Second, false},
		{"2m", 2 * time.Minute, false},
		{"1h30m", 90 * time.Minute, false},
		{"100ms", 100 * time.Millisecond, false},
		{"notaduration", 0, true},
	}

	for _, tc := range cases {
		yaml := "probes:\n  - name: x\n    url: u\n    interval: " + tc.input + "\n"
		path := writeTemp(t, yaml)
		cfg, err := config.Load(path)
		if tc.wantErr {
			if err == nil {
				t.Errorf("interval %q: want error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("interval %q: unexpected error: %v", tc.input, err)
			continue
		}
		if got := cfg.Probes[0].Interval.Duration; got != tc.want {
			t.Errorf("interval %q: got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}
