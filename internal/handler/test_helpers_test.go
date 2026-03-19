//go:build integration

package handler_test

import (
	"net/http"
	"sync"

	"github.com/zachbroad/nitrohook/internal/dispatch"
	"github.com/zachbroad/nitrohook/internal/model"
)

var registerOnce sync.Once

func registerTestDispatchers() {
	registerOnce.Do(func() {
		client := &http.Client{}
		dispatch.Register(model.ActionTypeWebhook, &dispatch.WebhookDispatcher{Client: client})
		dispatch.Register(model.ActionTypeJavascript, &dispatch.JavascriptDispatcher{})
		dispatch.Register(model.ActionTypeSlack, &dispatch.SlackDispatcher{Client: client})
		dispatch.Register(model.ActionTypeSMTP, &dispatch.SMTPDispatcher{})
		dispatch.Register(model.ActionTypeTwilio, &dispatch.TwilioDispatcher{Client: client})
	})
}
