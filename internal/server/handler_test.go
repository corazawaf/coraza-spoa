package server

import (
	"testing"

	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/negasus/haproxy-spoe-go/message"
	"github.com/negasus/haproxy-spoe-go/payload/kv"
	"github.com/negasus/haproxy-spoe-go/request"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	setApps(newAppManager())
	apps := getApps()
	assert.NoError(t, apps.Add(&config.Application{
		Name:          "test",
		ResponseCheck: true,
		Directives: `
		SecRuleEngine On
		SecRule ARGS "@contains test" "id:1,phase:2,deny,status:403"`,
	}))
	handler := &handler{}
	msg := &message.Message{
		Name: messageCorazaRequest,
		KV:   kv.NewKV(),
	}
	msg.KV.Add("app", "test")
	msg.KV.Add("id", "test_id")
	msg.KV.Add("headers", "Host: localhost\r\nContent-Type: application/json")
	messages := &message.Messages{
		msg,
	}

	req := request.Request{
		Messages: messages,
	}
	assert.NoError(t, handler.handler(&req))

	handler.Handler(&req)
	// now we look for interruptions ...

	msg = &message.Message{
		Name: messageCorazaResponse,
		KV:   kv.NewKV(),
	}
	msg.KV.Add("app", "test")
	msg.KV.Add("id", "test_id")
	msg.KV.Add("headers", "Content-Type: application/json")
	messages = &message.Messages{
		msg,
	}
	req = request.Request{
		Messages: messages,
	}
	assert.NoError(t, handler.handler(&req))
}
