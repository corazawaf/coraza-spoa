package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/negasus/haproxy-spoe-go/message"
)

type applicationRequest struct {
	app     string
	id      string
	srcIp   net.IP
	srcPort int64
	dstIp   net.IP
	dstPort int64
	method  string
	path    string
	query   string
	version string
	headers string
	body    []byte
}

func (r *applicationRequest) Fill(msg *message.Message) error {
	d := map[string]interface{}{
		"app":      &r.app,
		"id":       &r.id,
		"src-ip":   &r.srcIp,
		"src-port": &r.srcPort,
		"dst-ip":   &r.dstIp,
		"dst-port": &r.dstPort,
		"method":   &r.method,
		"path":     &r.path,
		"query":    &r.query,
		"version":  &r.version,
		"headers":  &r.headers,
		"body":     &r.body,
	}

	for _, v := range msg.KV.Data() {
		ptr, ok := d[v.Name]
		if !ok {
			return fmt.Errorf("unknown key %q", v.Name)
		}
		if v.Value == nil {
			continue
		}
		// now we overwrite the pointer
		switch p := ptr.(type) {
		case *string:
			*p = v.Value.(string)
		case *int64:
			*p = v.Value.(int64)
		case *net.IP:
			// we validate the interface is [4]byte
			*p = v.Value.(net.IP)
		case *[]byte:
			*p = v.Value.([]byte)
		// Add other types as necessary
		default:
			return fmt.Errorf("unsupported type for key %q", v.Name)
		}
	}
	return nil
}

func (r *applicationRequest) String() string {
	return fmt.Sprintf("app=%s id=%s src_ip=%s src_port=%d dst_ip=%s dst_port=%d method=%s path=%s query=%s version=%s headers=%s body=%s",
		r.app, r.id, r.srcIp, r.srcPort, r.dstIp, r.dstPort, r.method, r.path, r.query, r.version, r.headers, r.body)
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return &applicationRequest{
			app:     "",
			id:      "",
			srcIp:   nil,
			srcPort: 0,
			dstIp:   nil,
			dstPort: 0,
			method:  "",
			path:    "",
			query:   "",
			version: "",
			headers: "",
			body:    nil,
		}
	},
}
