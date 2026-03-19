package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zachbroad/nitrohook/internal/model"
)

type TwilioConfig struct {
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	From       string `json:"from"`
	To         string `json:"to"`
	BodyTmpl   string `json:"body_template,omitempty"`
}

type TwilioDispatcher struct {
	Client *http.Client
}

func (d *TwilioDispatcher) Validate(action *model.Action) error {
	cfg, err := parseConfig[TwilioConfig](action)
	if err != nil {
		return err
	}
	if cfg.AccountSID == "" {
		return fmt.Errorf("config.account_sid is required for twilio actions")
	}
	if cfg.AuthToken == "" {
		return fmt.Errorf("config.auth_token is required for twilio actions")
	}
	if cfg.From == "" {
		return fmt.Errorf("config.from is required for twilio actions")
	}
	if cfg.To == "" {
		return fmt.Errorf("config.to is required for twilio actions")
	}
	return nil
}

func (d *TwilioDispatcher) Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result {
	cfg, err := parseConfig[TwilioConfig](action)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	body := fmt.Sprintf("Webhook %s: %s", deliveryID, string(payload))
	if cfg.BodyTmpl != "" {
		body = cfg.BodyTmpl
	}

	// Truncate SMS body to 1600 chars (Twilio limit)
	if len(body) > 1600 {
		body = body[:1597] + "..."
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.AccountSID)

	form := url.Values{}
	form.Set("From", cfg.From)
	form.Set("To", cfg.To)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.AccountSID, cfg.AuthToken)

	resp, err := d.Client.Do(req)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyLen))
	respStr := string(respBody)
	statusCode := resp.StatusCode

	if statusCode >= 200 && statusCode < 300 {
		return Result{Success: true, ResponseStatus: &statusCode, ResponseBody: &respStr}
	}

	errMsg := fmt.Sprintf("Twilio HTTP %d: %s", statusCode, respStr)
	return Result{ResponseStatus: &statusCode, ResponseBody: &respStr, ErrorMessage: &errMsg}
}
