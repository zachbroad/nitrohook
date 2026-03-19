package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zachbroad/nitrohook/internal/model"
)

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
}

type SlackDispatcher struct {
	Client *http.Client
}

func (d *SlackDispatcher) Validate(action *model.Action) error {
	cfg, err := parseConfig[SlackConfig](action)
	if err != nil {
		return err
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("config.webhook_url is required for slack actions")
	}
	return nil
}

func (d *SlackDispatcher) Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result {
	cfg, err := parseConfig[SlackConfig](action)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	// Build Slack message with the webhook payload as a code block
	slackMsg := map[string]any{
		"text": fmt.Sprintf("Webhook delivery `%s`:\n```%s```", deliveryID, string(payload)),
	}
	if cfg.Channel != "" {
		slackMsg["channel"] = cfg.Channel
	}
	if cfg.Username != "" {
		slackMsg["username"] = cfg.Username
	}

	body, _ := json.Marshal(slackMsg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}
	req.Header.Set("Content-Type", "application/json")

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

	errMsg := fmt.Sprintf("Slack HTTP %d: %s", statusCode, respStr)
	return Result{ResponseStatus: &statusCode, ResponseBody: &respStr, ErrorMessage: &errMsg}
}
