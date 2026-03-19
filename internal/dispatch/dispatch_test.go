package dispatch

import (
	"encoding/json"
	"testing"

	"github.com/zachbroad/nitrohook/internal/model"
)

func TestParseConfigValid(t *testing.T) {
	action := &model.Action{
		Config: json.RawMessage(`{"webhook_url":"https://hooks.slack.com/test"}`),
	}
	cfg, err := parseConfig[SlackConfig](action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WebhookURL != "https://hooks.slack.com/test" {
		t.Fatalf("expected webhook_url to be parsed, got %q", cfg.WebhookURL)
	}
}

func TestParseConfigMissing(t *testing.T) {
	action := &model.Action{Config: nil}
	_, err := parseConfig[SlackConfig](action)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestParseConfigMalformed(t *testing.T) {
	action := &model.Action{Config: json.RawMessage(`{bad json`)}
	_, err := parseConfig[SlackConfig](action)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered type")
	}
}

func TestWebhookValidate(t *testing.T) {
	d := &WebhookDispatcher{}

	// Missing target_url
	err := d.Validate(&model.Action{Type: model.ActionTypeWebhook})
	if err == nil {
		t.Fatal("expected error for missing target_url")
	}

	// Empty target_url
	empty := ""
	err = d.Validate(&model.Action{Type: model.ActionTypeWebhook, TargetURL: &empty})
	if err == nil {
		t.Fatal("expected error for empty target_url")
	}

	// Valid
	url := "https://example.com/hook"
	err = d.Validate(&model.Action{Type: model.ActionTypeWebhook, TargetURL: &url})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSlackValidate(t *testing.T) {
	d := &SlackDispatcher{}

	// Missing config
	err := d.Validate(&model.Action{Type: model.ActionTypeSlack})
	if err == nil {
		t.Fatal("expected error for missing config")
	}

	// Missing webhook_url
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSlack,
		Config: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for missing webhook_url")
	}

	// Valid
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSlack,
		Config: json.RawMessage(`{"webhook_url":"https://hooks.slack.com/test"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSMTPValidate(t *testing.T) {
	d := &SMTPDispatcher{}

	// Missing config
	err := d.Validate(&model.Action{Type: model.ActionTypeSMTP})
	if err == nil {
		t.Fatal("expected error for missing config")
	}

	// Missing host
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSMTP,
		Config: json.RawMessage(`{"from":"a@b.com","to":"c@d.com","port":587}`),
	})
	if err == nil {
		t.Fatal("expected error for missing host")
	}

	// Missing from
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSMTP,
		Config: json.RawMessage(`{"host":"smtp.example.com","to":"c@d.com","port":587}`),
	})
	if err == nil {
		t.Fatal("expected error for missing from")
	}

	// Missing to
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSMTP,
		Config: json.RawMessage(`{"host":"smtp.example.com","from":"a@b.com","port":587}`),
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}

	// Missing port
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSMTP,
		Config: json.RawMessage(`{"host":"smtp.example.com","from":"a@b.com","to":"c@d.com"}`),
	})
	if err == nil {
		t.Fatal("expected error for missing port")
	}

	// Valid
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeSMTP,
		Config: json.RawMessage(`{"host":"smtp.example.com","from":"a@b.com","to":"c@d.com","port":587}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTwilioValidate(t *testing.T) {
	d := &TwilioDispatcher{}

	// Missing config
	err := d.Validate(&model.Action{Type: model.ActionTypeTwilio})
	if err == nil {
		t.Fatal("expected error for missing config")
	}

	// Missing account_sid
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"auth_token":"tok","from":"+1234","to":"+5678"}`),
	})
	if err == nil {
		t.Fatal("expected error for missing account_sid")
	}

	// Missing auth_token
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"sid","from":"+1234","to":"+5678"}`),
	})
	if err == nil {
		t.Fatal("expected error for missing auth_token")
	}

	// Missing from
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"sid","auth_token":"tok","to":"+5678"}`),
	})
	if err == nil {
		t.Fatal("expected error for missing from")
	}

	// Missing to
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"sid","auth_token":"tok","from":"+1234"}`),
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}

	// Valid
	err = d.Validate(&model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"sid","auth_token":"tok","from":"+1234","to":"+5678"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
