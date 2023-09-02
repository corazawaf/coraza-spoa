package server

import (
	"fmt"
	"strings"
)

func readHeaders(headers string, callback func(key string, value string)) error {
	hs := strings.Split(headers, "\n")
	for _, header := range hs {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}

		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid header: %q", header)
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		callback(key, value)
	}
	return nil
}
