package dispatch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
)

type JavascriptDispatcher struct{}

func (d *JavascriptDispatcher) Validate(action *model.Action) error {
	if action.ScriptBody == nil || *action.ScriptBody == "" {
		return fmt.Errorf("script_body is required for javascript actions")
	}
	return script.ValidateAction(*action.ScriptBody)
}

func (d *JavascriptDispatcher) Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result {
	if action.ScriptBody == nil || *action.ScriptBody == "" {
		errMsg := "javascript action has no script_body"
		return Result{ErrorMessage: &errMsg}
	}

	var payloadMap map[string]any
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		errMsg := fmt.Sprintf("failed to unmarshal payload: %v", err)
		return Result{ErrorMessage: &errMsg}
	}

	var headersMap map[string]string
	if err := json.Unmarshal(headers, &headersMap); err != nil {
		errMsg := fmt.Sprintf("failed to unmarshal headers: %v", err)
		return Result{ErrorMessage: &errMsg}
	}

	result, err := script.RunAction(*action.ScriptBody, payloadMap, headersMap)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	return Result{Success: true, ResponseBody: &result}
}
