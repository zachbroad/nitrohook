package script

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
)

const (
	maxScriptSize = 64 * 1024 // 64KB
	execTimeout   = 500 * time.Millisecond
)

var (
	ErrScriptTooLarge    = errors.New("script exceeds 64KB limit")
	ErrScriptTimeout     = errors.New("script execution timed out")
	ErrNoTransform       = errors.New("script must define a 'transform' function")
	ErrNoProcess         = errors.New("script must define a 'process' function")
	ErrNoActionTransform = errors.New("script must define a 'transform' function")
)

// ActionRef is a lightweight action reference passed into/out of scripts.
type ActionRef struct {
	ID        uuid.UUID `json:"id"`
	TargetURL string    `json:"target_url"`
}

// TransformInput is the data passed to the transform function.
type TransformInput struct {
	Payload map[string]any    `json:"payload"`
	Headers map[string]string `json:"headers"`
	Actions []ActionRef       `json:"actions"`
}

// TransformResult is the output of the transform function.
type TransformResult struct {
	Payload map[string]any    `json:"payload"`
	Headers map[string]string `json:"headers"`
	Actions []ActionRef       `json:"actions"`
	Dropped bool              `json:"dropped"`
}

// runVM compiles scriptBody, calls the named function with arg, and returns the JS value.
// missingErr is returned when the function is not found/not callable.
// Handles size check, timeout, panic recovery, and compilation errors.
func runVM(scriptBody, fnName string, missingErr error, arg any) (ret goja.Value, err error) {
	if len(scriptBody) > maxScriptSize {
		return nil, ErrScriptTooLarge
	}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*goja.InterruptedError); ok {
				ret = nil
				err = ErrScriptTimeout
			} else {
				ret = nil
				err = fmt.Errorf("script panic: %v", r)
			}
		}
	}()

	vm := goja.New()

	timer := time.AfterFunc(execTimeout, func() {
		vm.Interrupt("timeout")
	})
	defer timer.Stop()

	if _, err = vm.RunString(scriptBody); err != nil {
		return nil, fmt.Errorf("script compilation error: %w", err)
	}

	fn := vm.Get(fnName)
	if fn == nil || fn == goja.Undefined() || fn == goja.Null() {
		return nil, missingErr
	}
	callable, ok := goja.AssertFunction(fn)
	if !ok {
		return nil, missingErr
	}

	ret, err = callable(goja.Undefined(), vm.ToValue(arg))
	if err != nil {
		var interrupted *goja.InterruptedError
		if errors.As(err, &interrupted) {
			return nil, ErrScriptTimeout
		}
		return nil, fmt.Errorf("script execution error: %w", err)
	}
	return ret, nil
}

// validateScript checks that the script compiles and exports the named function.
func validateScript(scriptBody, fnName string, missingErr error) error {
	if len(scriptBody) > maxScriptSize {
		return ErrScriptTooLarge
	}
	vm := goja.New()
	if _, err := vm.RunString(scriptBody); err != nil {
		return fmt.Errorf("script compilation error: %w", err)
	}
	fn := vm.Get(fnName)
	if fn == nil || fn == goja.Undefined() || fn == goja.Null() {
		return missingErr
	}
	if _, ok := goja.AssertFunction(fn); !ok {
		return missingErr
	}
	return nil
}

// Validate checks that the script compiles and exports a 'transform' function.
func Validate(scriptBody string) error {
	return validateScript(scriptBody, "transform", ErrNoTransform)
}

// Run executes the transform function with the given input.
// Returns nil result with Dropped=true if the script returns null/undefined.
func Run(scriptBody string, input TransformInput) (*TransformResult, error) {
	eventObj := map[string]any{
		"payload": input.Payload,
		"headers": input.Headers,
	}
	actionsForJS := make([]map[string]any, len(input.Actions))
	for i, a := range input.Actions {
		actionsForJS[i] = map[string]any{
			"id":         a.ID.String(),
			"target_url": a.TargetURL,
		}
	}
	eventObj["actions"] = actionsForJS

	ret, err := runVM(scriptBody, "transform", ErrNoTransform, eventObj)
	if err != nil {
		return nil, err
	}

	if ret == nil || ret == goja.Undefined() || ret == goja.Null() {
		return &TransformResult{Dropped: true}, nil
	}

	exported := ret.Export()
	jsonBytes, err := json.Marshal(exported)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal script result: %w", err)
	}

	var raw struct {
		Payload map[string]any `json:"payload"`
		Headers map[string]any `json:"headers"`
		Actions []struct {
			ID        string `json:"id"`
			TargetURL string `json:"target_url"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal script result: %w", err)
	}

	headers := make(map[string]string, len(raw.Headers))
	for k, v := range raw.Headers {
		headers[k] = fmt.Sprintf("%v", v)
	}

	actions := make([]ActionRef, 0, len(raw.Actions))
	for _, a := range raw.Actions {
		id, err := uuid.Parse(a.ID)
		if err != nil {
			continue
		}
		actions = append(actions, ActionRef{ID: id, TargetURL: a.TargetURL})
	}

	return &TransformResult{
		Payload: raw.Payload,
		Headers: headers,
		Actions: actions,
	}, nil
}

// ValidateAction checks that the script compiles and exports a 'process' function.
func ValidateAction(scriptBody string) error {
	return validateScript(scriptBody, "process", ErrNoProcess)
}

// RunAction executes a per-action JS script's process(event) function.
// Returns the result as a JSON string.
func RunAction(scriptBody string, payload map[string]any, headers map[string]string) (string, error) {
	eventObj := map[string]any{
		"payload": payload,
		"headers": headers,
	}

	ret, err := runVM(scriptBody, "process", ErrNoProcess, eventObj)
	if err != nil {
		return "", err
	}

	if ret == nil || ret == goja.Undefined() || ret == goja.Null() {
		return "null", nil
	}

	jsonBytes, err := json.Marshal(ret.Export())
	if err != nil {
		return "", fmt.Errorf("failed to marshal action script result: %w", err)
	}
	return string(jsonBytes), nil
}

// ActionTransformResult is the output of a per-action transform function.
type ActionTransformResult struct {
	Payload map[string]any    `json:"payload"`
	Headers map[string]string `json:"headers"`
	Skipped bool              `json:"skipped"`
}

// ValidateActionTransform checks that the script compiles and exports a 'transform' function.
func ValidateActionTransform(scriptBody string) error {
	return validateScript(scriptBody, "transform", ErrNoActionTransform)
}

// RunActionTransform executes a per-action transform(event) function.
// Returns nil with Skipped=true if the script returns null/undefined (skip this action).
func RunActionTransform(scriptBody string, payload map[string]any, headers map[string]string) (*ActionTransformResult, error) {
	eventObj := map[string]any{
		"payload": payload,
		"headers": headers,
	}

	ret, err := runVM(scriptBody, "transform", ErrNoActionTransform, eventObj)
	if err != nil {
		return nil, err
	}

	if ret == nil || ret == goja.Undefined() || ret == goja.Null() {
		return &ActionTransformResult{Skipped: true}, nil
	}

	jsonBytes, err := json.Marshal(ret.Export())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transform result: %w", err)
	}

	var raw struct {
		Payload map[string]any `json:"payload"`
		Headers map[string]any `json:"headers"`
	}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transform result: %w", err)
	}

	hdrs := make(map[string]string, len(raw.Headers))
	for k, v := range raw.Headers {
		hdrs[k] = fmt.Sprintf("%v", v)
	}

	return &ActionTransformResult{
		Payload: raw.Payload,
		Headers: hdrs,
	}, nil
}
