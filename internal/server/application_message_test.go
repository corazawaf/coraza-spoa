package server

import (
	"net"
	"testing"

	"github.com/negasus/haproxy-spoe-go/message"
	"github.com/negasus/haproxy-spoe-go/payload/kv"
	"github.com/stretchr/testify/assert"
)

type test struct {
	Name   string `spoa:"name"`
	Number int    `spoa:"number"`
	IP     net.IP `spoa:"ip"`
}

func TestApplicationMessage(t *testing.T) {
	msg := message.Message{
		Name: "test",
		KV:   kv.NewKV(),
	}
	msg.KV.Add("name", "test")
	msg.KV.Add("number", int64(1234))
	msg.KV.Add("ip", net.ParseIP("127.0.0.1"))
	tt := &test{}
	assert.NoError(t, unmarshalMessage(&msg, tt))
	assert.Equal(t, "test", tt.Name)
	assert.Equal(t, 1234, tt.Number)
	assert.Equal(t, net.ParseIP("127.0.0.1"), tt.IP)
}

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
	assert.NoError(t, unmarshalMessage(msg, req))
	assert.Equal(t, "test", req.App)

	t.Run("headers should be parsed", func(t *testing.T) {
		assert.NoError(t, readHeaders(req.Headers, func(key, value string) {
			assert.Equal(t, "Host", key)
			assert.Equal(t, "localhost", value)
		}))
	})
}
