package server

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/negasus/haproxy-spoe-go/message"
)

func unmarshalMessage(msg *message.Message, out interface{}) error {
	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("expected pointer to struct")
	}
	val = val.Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		tag := field.Tag.Get("spoa")
		if tag == "" {
			continue
		}
		name := tag
		fieldValue := val.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		kv, ok := msg.KV.Get(name)
		if !ok {
			continue
		}
		if kv == nil {
			continue
		}
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(kv.(string))
		case reflect.Int:
			fieldValue.SetInt(kv.(int64))
		case reflect.Int64:
			fieldValue.SetInt(kv.(int64))
		case reflect.Slice:
			if fieldValue.Type().String() == "net.IP" {
				fieldValue.Set(reflect.ValueOf(kv.(net.IP)))
			} else {
				fieldValue.Set(reflect.ValueOf(kv))
			}
		default:
			return fmt.Errorf("unsupported type: %s", fieldValue.Type().String())
		}
	}
	return nil
}

type applicationRequest struct {
	App     string `spoa:"app"`
	ID      string `spoa:"id"`
	SrcIp   net.IP `spoa:"src-ip"`
	SrcPort int64  `spoa:"src-port"`
	DstIp   net.IP `spoa:"dst-ip"`
	DstPort int64  `spoa:"dst-port"`
	Method  string `spoa:"method"`
	Path    string `spoa:"path"`
	Query   string `spoa:"query"`
	Version string `spoa:"version"`
	Headers string `spoa:"headers"`
	Body    []byte `spoa:"body"`
}

func (r *applicationRequest) String() string {
	return fmt.Sprintf("app=%s id=%s src_ip=%s src_port=%d dst_ip=%s dst_port=%d method=%s path=%s query=%s version=%s headers=%s body=%s",
		r.App, r.ID, r.SrcIp, r.SrcPort, r.DstIp, r.DstPort, r.Method, r.Path, r.Query, r.Version, r.Headers, r.Body)
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return &applicationRequest{
			App:     "",
			ID:      "",
			SrcIp:   nil,
			SrcPort: 0,
			DstIp:   nil,
			DstPort: 0,
			Method:  "",
			Path:    "",
			Query:   "",
			Version: "",
			Headers: "",
			Body:    nil,
		}
	},
}

type applicationResponse struct {
	App     string `spoa:"app"`
	ID      string `spoa:"id"`
	Version string `spoa:"version"`
	Status  int    `spoa:"status"`
	Headers string `spoa:"headers"`
	Body    []byte `spoa:"body"`
}

var responsePool = sync.Pool{
	New: func() interface{} {
		return &applicationResponse{
			App:     "",
			ID:      "",
			Version: "",
			Status:  0,
			Headers: "",
			Body:    []byte{},
		}
	},
}
