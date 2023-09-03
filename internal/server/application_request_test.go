package server

import (
	"net"
	"testing"

	"github.com/negasus/haproxy-spoe-go/message"
	"github.com/negasus/haproxy-spoe-go/payload/kv"
	"github.com/stretchr/testify/assert"
)

func TestApplicationRequest(t *testing.T) {
	msg := &message.Message{
		Name: messageCorazaRequest,
		KV:   kv.NewKV(),
	}
	msg.KV.Add("app", "test")
	msg.KV.Add("id", "test_id")
	msg.KV.Add("headers", "Host: localhost")
	msg.KV.Add("src-ip", net.ParseIP("192.168.1.1"))
	msg.KV.Add("src-port", int64(1234))
	msg.KV.Add("dst-ip", net.ParseIP("1.1.1.1"))
	msg.KV.Add("dst-port", int64(4321))
	msg.KV.Add("method", "GET")
	msg.KV.Add("path", "/test")
	msg.KV.Add("query", "test=1")
	msg.KV.Add("version", "1.1")
	msg.KV.Add("body", []byte("test=test"))
	req := requestPool.Get().(*applicationRequest)
	defer requestPool.Put(req)
	assert.NoError(t, req.Fill(msg))

	t.Run("invalid message key should fail", func(t *testing.T) {
		msg.KV.Add("false_error", "test")
		req2 := requestPool.Get().(*applicationRequest)
		defer requestPool.Put(req2)
		assert.Error(t, req2.Fill(msg))
	})

	assert.Equal(t, "test", req.app)

	t.Run("headers should be parsed", func(t *testing.T) {
		assert.NoError(t, readHeaders(req.headers, func(key, value string) {
			assert.Equal(t, "Host", key)
			assert.Equal(t, "localhost", value)
		}))
	})
}
