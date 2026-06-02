// Package prober runs HTTP health checks against configured endpoints
// and reports state changes through a Notifier.
package prober

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cactus/config"
	"cactus/notifier"
)

type state struct {
	up      bool
	checked bool
}

// Run executes the probe loop for cfg until stop is closed.
// It alerts n on the first failure and on each down→up recovery.
// A deferred recover ensures a panic in one probe never kills the process.
func Run(cfg *config.ProbeConfig, n notifier.Notifier, stop <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[%s] recovered from panic: %v", cfg.Name, r)
		}
	}()

	client := &http.Client{Timeout: 10 * time.Second}

	interval := cfg.Interval.Duration
	if interval <= 0 {
		interval = 60 * time.Second
	}

	var s state
	check(cfg, client, &s, n)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			check(cfg, client, &s, n)
		case <-stop:
			return
		}
	}
}

func check(cfg *config.ProbeConfig, client *http.Client, s *state, n notifier.Notifier) {
	method := cfg.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, cfg.URL, nil)
	if err != nil {
		markDown(cfg, s, n, fmt.Errorf("build request: %w", err))
		return
	}

	if cfg.Auth != nil {
		raw := cfg.Auth.Username + ":" + cfg.Auth.Password
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		markDown(cfg, s, n, fmt.Errorf("request: %w", err))
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	expected := cfg.ExpectedStatus
	if expected == 0 {
		expected = 200
	}

	if resp.StatusCode != expected {
		markDown(cfg, s, n, fmt.Errorf("got status %d, want %d", resp.StatusCode, expected))
		return
	}

	markUp(cfg, s, n)
}

func markDown(cfg *config.ProbeConfig, s *state, n notifier.Notifier, err error) {
	log.Printf("[%s] FAIL: %v", cfg.Name, err)
	wasUp := s.up
	s.up = false
	// Alert on first check or on transition from up → down.
	if !s.checked || wasUp {
		n.Alert(cfg.Name, cfg.URL, false, err.Error())
	}
	s.checked = true
}

func markUp(cfg *config.ProbeConfig, s *state, n notifier.Notifier) {
	log.Printf("[%s] OK", cfg.Name)
	wasUp := s.up
	s.up = true
	// Alert only on recovery; suppress steady-state OK.
	if s.checked && !wasUp {
		n.Alert(cfg.Name, cfg.URL, true, "")
	}
	s.checked = true
}
