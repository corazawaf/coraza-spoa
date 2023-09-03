package server

import (
	"fmt"
	"sync"

	"github.com/negasus/haproxy-spoe-go/message"
)

type applicationResponse struct {
	app     string
	id      string
	version string
	status  int
	headers string
	body    []byte
}

func (r *applicationResponse) Fill(msg *message.Message) error {
	d := map[string]interface{}{
		"app":     &r.app,
		"id":      &r.id,
		"version": &r.version,
		"status":  &r.status,
		"headers": &r.headers,
		"body":    &r.body,
	}

	for _, v := range msg.KV.Data() {
		ptr, ok := d[v.Name]
		if !ok {
			return fmt.Errorf("unknown key %q", v.Name)
		}
		// now we overwrite the pointer
		switch p := ptr.(type) {
		case *string:
			*p = v.Value.(string)
		case *int:
			*p = v.Value.(int)
		case *[]byte:
			*p = v.Value.([]byte)
		// Add other types as necessary
		default:
			return fmt.Errorf("unsupported type for key %q", v.Name)
		}
	}
	return nil
}

var responsePool = sync.Pool{
	New: func() interface{} {
		return &applicationResponse{
			app:     "",
			id:      "",
			version: "",
			status:  0,
			headers: "",
			body:    []byte{},
		}
	},
}
