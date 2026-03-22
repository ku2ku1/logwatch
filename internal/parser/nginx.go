package parser

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	IP          string
	Time        time.Time
	Method      string
	Path        string
	Protocol    string
	StatusCode  int
	BytesSent   int64
	Referer     string
	UserAgent   string
	ResponseTime float64
	Raw         string
}

// Nginx combined log format
var nginxRegex = regexp.MustCompile(
	`^(\S+)\s+-\s+(\S+)\s+\[([^\]]+)\]\s+"(\S+)\s+(\S+)\s+(\S+)"\s+(\d+)\s+(\d+)\s+"([^"]*)"\s+"([^"]*)"(?:\s+(\S+))?`,
)

var timeLayout = "02/Jan/2006:15:04:05 -0700"

func ParseNginxLine(line string) (*LogEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	matches := nginxRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("no match: %s", line[:min(len(line), 80)])
	}

	t, err := time.Parse(timeLayout, matches[3])
	if err != nil {
		return nil, fmt.Errorf("time parse: %w", err)
	}

	status, _ := strconv.Atoi(matches[7])
	bytes, _ := strconv.ParseInt(matches[8], 10, 64)

	// Response time (optional — $request_time)
	var respTime float64
	if matches[11] != "" && matches[11] != "-" {
		respTime, _ = strconv.ParseFloat(matches[11], 64)
	}

	// Clean path — remove query string for grouping
	rawPath := matches[5]
	cleanPath := rawPath
	if u, err := url.Parse(rawPath); err == nil {
		cleanPath = u.Path
	}

	return &LogEntry{
		IP:           matches[1],
		Time:         t,
		Method:       matches[4],
		Path:         cleanPath,
		Protocol:     matches[6],
		StatusCode:   status,
		BytesSent:    bytes,
		Referer:      matches[9],
		UserAgent:    matches[10],
		ResponseTime: respTime,
		Raw:          line,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
