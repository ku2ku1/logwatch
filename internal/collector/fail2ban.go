package collector

import (
	"bufio"
	"os"
	"regexp"
	"time"
)

type BanEvent struct {
	Time    time.Time
	Jail    string
	IP      string
	Action  string // Ban, Unban, Found
}

var (
	banRegex   = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}),\d+ fail2ban\.actions\s+\[.*?\]: (NOTICE|WARNING)\s+\[(\w[\w-]*)\] (Ban|Unban|Found) (\S+)`)
	timeLayout = "2006-01-02 15:04:05"
)

func ParseFail2banLog(path string, since time.Time) ([]BanEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []BanEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		matches := banRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		t, err := time.ParseInLocation(timeLayout, matches[1], time.Local)
		if err != nil || t.Before(since) {
			continue
		}
		events = append(events, BanEvent{
			Time:   t,
			Jail:   matches[3],
			Action: matches[4],
			IP:     matches[5],
		})
	}
	return events, scanner.Err()
}

type Fail2banStats struct {
	TotalBans    int64
	ActiveBans   int64
	TotalJails   int
	TopJails     map[string]int64
	RecentBans   []BanEvent
	BannedIPs    []string
}

func GetFail2banStats(logPath string, since time.Time) (*Fail2banStats, error) {
	events, err := ParseFail2banLog(logPath, since)
	if err != nil {
		return nil, err
	}

	stats := &Fail2banStats{
		TopJails: make(map[string]int64),
	}

	activeBans := make(map[string]bool)
	jails := make(map[string]bool)

	for _, e := range events {
		jails[e.Jail] = true
		switch e.Action {
		case "Ban":
			stats.TotalBans++
			stats.TopJails[e.Jail]++
			activeBans[e.IP] = true
		case "Unban":
			delete(activeBans, e.IP)
		}
	}

	stats.ActiveBans = int64(len(activeBans))
	stats.TotalJails = len(jails)
	for ip := range activeBans {
		stats.BannedIPs = append(stats.BannedIPs, ip)
	}

	// Last 20 ban events
	if len(events) > 20 {
		stats.RecentBans = events[len(events)-20:]
	} else {
		stats.RecentBans = events
	}

	return stats, nil
}
