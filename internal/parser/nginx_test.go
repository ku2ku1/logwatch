package parser

import (
	"fmt"
	"testing"
)

func TestParseNginxLine(t *testing.T) {
	lines := []string{
		`::1 - - [21/Mar/2026:11:44:59 -0400] "GET / HTTP/1.1" 200 10703 "-" "curl/8.18.0"`,
		`::1 - - [21/Mar/2026:11:45:09 -0400] "GET /about HTTP/1.1" 404 146 "-" "curl/8.18.0"`,
	}
	for _, line := range lines {
		entry, err := ParseNginxLine(line)
		if err != nil {
			t.Errorf("ERROR: %v", err)
		} else {
			fmt.Printf("OK: IP=%s Method=%s Path=%s Status=%d\n",
				entry.IP, entry.Method, entry.Path, entry.StatusCode)
		}
	}
}
