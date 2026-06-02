package notifier

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// TelegramConfig holds credentials for the Telegram Bot API.
type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// Telegram sends alerts via the Telegram Bot API.
type Telegram struct {
	cfg         TelegramConfig
	client      *http.Client
	endpointFmt string // format string for the sendMessage URL; %s is replaced by BotToken
}

const defaultEndpointFmt = "https://api.telegram.org/bot%s/sendMessage"

// NewTelegram returns a Telegram notifier configured with cfg.
func NewTelegram(cfg TelegramConfig) *Telegram {
	return NewTelegramWithEndpoint(cfg, defaultEndpointFmt)
}

// NewTelegramWithEndpoint returns a Telegram notifier with a custom endpoint
// format string. The %s placeholder is replaced with BotToken. Intended for tests.
func NewTelegramWithEndpoint(cfg TelegramConfig, endpointFmt string) *Telegram {
	return &Telegram{
		cfg:         cfg,
		client:      &http.Client{Timeout: 15 * time.Second},
		endpointFmt: endpointFmt,
	}
}

// Alert implements Notifier by posting a message to the configured Telegram chat.
func (t *Telegram) Alert(name, targetURL string, up bool, errMsg string) {
	var header string
	if up {
		header = "[RESOLVED] " + name + " is back up"
	} else {
		header = "[FIRING] " + name + " is down"
	}

	msg := header + "\nURL: " + targetURL
	if !up && errMsg != "" {
		msg += "\nError: " + errMsg
	}

	if err := t.send(msg); err != nil {
		log.Printf("telegram: send failed: %v", err)
	}
}

func (t *Telegram) send(text string) error {
	endpoint := fmt.Sprintf(t.endpointFmt, t.cfg.BotToken)

	params := url.Values{}
	params.Set("chat_id", t.cfg.ChatID)
	params.Set("text", text)

	resp, err := t.client.PostForm(endpoint, params)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned %d", resp.StatusCode)
	}
	return nil
}
