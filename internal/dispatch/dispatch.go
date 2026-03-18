package dispatch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zachbroad/nitrohook/internal/model"
)

// Result captures the outcome of dispatching to an action.
type Result struct {
	Success        bool
	ResponseStatus *int
	ResponseBody   *string
	ErrorMessage   *string
}

// Dispatcher handles delivery to a specific action type.
type Dispatcher interface {
	Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result
	Validate(action *model.Action) error
}

var registry = map[model.ActionType]Dispatcher{}

// Register adds a dispatcher for an action type.
func Register(actionType model.ActionType, d Dispatcher) {
	registry[actionType] = d
}

// Get returns the dispatcher for an action type.
func Get(actionType model.ActionType) (Dispatcher, error) {
	d, ok := registry[actionType]
	if !ok {
		return nil, fmt.Errorf("no dispatcher registered for action type %q", actionType)
	}
	return d, nil
}

// Types returns all registered action type names.
func Types() []model.ActionType {
	types := make([]model.ActionType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
