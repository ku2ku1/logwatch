package collector

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"time"
)

type UFWEvent struct {
	Time     time.Time
	Action   string // BLOCK, ALLOW
	SrcIP    string
	DstIP    string
	SrcPort  string
	DstPort  string
	Proto    string
}

var ufwRegex = regexp.MustCompile(
	`(\w{3}\s+\d+\s+\d{2}:\d{2}:\d{2}).*UFW (BLOCK|ALLOW|AUDIT).*SRC=(\S+).*DST=(\S+).*PROTO=(\S+)(?:.*SPT=(\d+))?(?:.*DPT=(\d+))?`,
)

func ParseUFWLog(path string, since time.Time) ([]UFWEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	year := time.Now().Year()
	var events []UFWEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "UFW") {
			continue
		}
		matches := ufwRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		t, err := time.ParseInLocation("2006 Jan  2 15:04:05",
			strings.TrimSpace(strings.Join([]string{string(rune(year + 48*1000)), matches[1]}, " ")),
			time.Local)
		if err != nil {
			// Simpler parse
			t = time.Now()
		}
		if t.Before(since) {
			continue
		}
		events = append(events, UFWEvent{
			Time:    t,
			Action:  matches[2],
			SrcIP:   matches[3],
			DstIP:   matches[4],
			Proto:   matches[5],
			SrcPort: matches[6],
			DstPort: matches[7],
		})
	}
	return events, scanner.Err()
}

type UFWStats struct {
	TotalBlocked  int64
	TotalAllowed  int64
	TopBlockedIPs map[string]int64
	TopPorts      map[string]int64
}

func GetUFWStats(logPath string, since time.Time) (*UFWStats, error) {
	events, err := ParseUFWLog(logPath, since)
	if err != nil {
		return nil, err
	}
	stats := &UFWStats{
		TopBlockedIPs: make(map[string]int64),
		TopPorts:      make(map[string]int64),
	}
	for _, e := range events {
		switch e.Action {
		case "BLOCK":
			stats.TotalBlocked++
			stats.TopBlockedIPs[e.SrcIP]++
			if e.DstPort != "" {
				stats.TopPorts[e.DstPort]++
			}
		case "ALLOW":
			stats.TotalAllowed++
		}
	}
	return stats, nil
}
