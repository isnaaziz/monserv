package notifier

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	gomail "gopkg.in/gomail.v2"
)

type Notifier interface {
	Send(subject, body string) error
	Name() string
}

type Multi struct {
	list []Notifier
}

func (m Multi) Send(subject, body string) error {
	var errs []string
	for _, n := range m.list {
		if err := n.Send(subject, body); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", n.Name(), err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (m Multi) Name() string { return "multi" }

// Email SMTP notifier using gomail
type Email struct {
	Host string
	Port int
	From string
	Pass string
	To   []string
}

func (e Email) Name() string { return "email" }

func (e Email) Send(subject, body string) error {
	if e.Host == "" || e.From == "" || len(e.To) == 0 {
		return fmt.Errorf("email not configured")
	}
	msg := gomail.NewMessage()
	msg.SetHeader("From", e.From)
	msg.SetHeader("To", e.To...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)
	d := gomail.NewDialer(e.Host, e.Port, e.From, e.Pass)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	return d.DialAndSend(msg)
}

// Slack webhook notifier
type Slack struct{ WebhookURL string }

func (s Slack) Name() string { return "slack" }

func (s Slack) Send(subject, body string) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack not configured")
	}
	payload := map[string]string{"text": fmt.Sprintf("*%s*\n%s", subject, body)}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook status %d", resp.StatusCode)
	}
	return nil
}

// Telegram bot notifier
type Telegram struct {
	BotToken string
	ChatID   string
}

func (t Telegram) Name() string { return "telegram" }

func (t Telegram) Send(subject, body string) error {
	if t.BotToken == "" || t.ChatID == "" {
		return fmt.Errorf("telegram not configured")
	}
	text := fmt.Sprintf("%s\n%s", subject, body)
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	params := map[string]string{"chat_id": t.ChatID, "text": text}
	b, _ := json.Marshal(params)
	resp, err := http.Post(api, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	return nil
}

// FromEnv builds a Multi notifier from environment variables
// EMAIL_SMTP_HOST, EMAIL_SMTP_PORT, EMAIL_FROM, EMAIL_PASSWORD, EMAIL_TO
// SLACK_WEBHOOK_URL
// TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID
func FromEnv() Notifier {
	var list []Notifier
	// Email
	if host := os.Getenv("EMAIL_SMTP_HOST"); host != "" {
		port := 587
		if v := os.Getenv("EMAIL_SMTP_PORT"); v != "" {
			fmt.Sscanf(v, "%d", &port)
		}
		to := []string{}
		if v := os.Getenv("EMAIL_TO"); v != "" {
			for _, s := range strings.Split(v, ",") {
				if t := strings.TrimSpace(s); t != "" {
					to = append(to, t)
				}
			}
		}
		list = append(list, Email{
			Host: host,
			Port: port,
			From: os.Getenv("EMAIL_FROM"),
			Pass: os.Getenv("EMAIL_PASSWORD"),
			To:   to,
		})
	}
	// Slack
	if v := os.Getenv("SLACK_WEBHOOK_URL"); v != "" {
		list = append(list, Slack{WebhookURL: v})
	}
	// Telegram
	if tok := os.Getenv("TELEGRAM_BOT_TOKEN"); tok != "" {
		list = append(list, Telegram{BotToken: tok, ChatID: os.Getenv("TELEGRAM_CHAT_ID")})
	}
	// Kafka
	if k, err := NewKafkaFromEnv(); err != nil {
		_ = err
	} else if k != nil {
		list = append(list, k)
	}
	if len(list) == 0 {
		return noop{}
	}
	return Multi{list: list}
}

type noop struct{}

func (n noop) Send(subject, body string) error { return nil }
func (n noop) Name() string                    { return "noop" }

// CooldownLimiter limits same subject alerts within a window
type CooldownLimiter struct {
	Inner    Notifier
	Cooldown time.Duration
	last     map[string]time.Time
	mu       sync.Mutex
}

func NewCooldown(inner Notifier, d time.Duration) *CooldownLimiter {
	return &CooldownLimiter{Inner: inner, Cooldown: d, last: map[string]time.Time{}}
}

func (c *CooldownLimiter) Send(subject, body string) error {
	now := time.Now()

	// Fast path under lock: check & set tentative send time
	c.mu.Lock()
	if t, ok := c.last[subject]; ok && now.Sub(t) < c.Cooldown {
		c.mu.Unlock()
		return nil
	}
	// set tentative timestamp to prevent duplicate sends from concurrent goroutines
	c.last[subject] = now
	c.mu.Unlock()

	// Send without holding lock
	if err := c.Inner.Send(subject, body); err != nil {
		// revert on failure so next attempt is allowed
		c.mu.Lock()
		delete(c.last, subject)
		c.mu.Unlock()
		return err
	}
	return nil
}

func (c *CooldownLimiter) Name() string { return "cooldown(" + c.Inner.Name() + ")" }
