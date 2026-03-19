package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"

	"github.com/zachbroad/nitrohook/internal/model"
)

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
}

type SMTPDispatcher struct{}

func (d *SMTPDispatcher) Validate(action *model.Action) error {
	cfg, err := parseConfig[SMTPConfig](action)
	if err != nil {
		return err
	}
	if cfg.Host == "" {
		return fmt.Errorf("config.host is required for smtp actions")
	}
	if cfg.From == "" {
		return fmt.Errorf("config.from is required for smtp actions")
	}
	if cfg.To == "" {
		return fmt.Errorf("config.to is required for smtp actions")
	}
	if cfg.Port == 0 {
		return fmt.Errorf("config.port is required for smtp actions")
	}
	return nil
}

func (d *SMTPDispatcher) Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result {
	cfg, err := parseConfig[SMTPConfig](action)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	subject := cfg.Subject
	if subject == "" {
		subject = fmt.Sprintf("Webhook delivery %s", deliveryID)
	}

	var body []byte
	indented, err := json.MarshalIndent(json.RawMessage(payload), "", "  ")
	if err != nil {
		body = payload
	} else {
		body = indented
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nDelivery ID: %s\n\n%s",
		cfg.From, cfg.To, subject, deliveryID, string(body))

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	if err := smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, []byte(msg)); err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	successMsg := fmt.Sprintf("sent to %s", cfg.To)
	return Result{Success: true, ResponseBody: &successMsg}
}
