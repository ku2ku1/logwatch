package analyzer

import (
	"regexp"
	"strings"
)

type ThreatType string

const (
	ThreatSQLi       ThreatType = "sql_injection"
	ThreatXSS        ThreatType = "xss"
	ThreatPathTraversal ThreatType = "path_traversal"
	ThreatScannerBot ThreatType = "scanner_bot"
	ThreatBruteForce ThreatType = "brute_force"
)

type Threat struct {
	Type        ThreatType
	Severity    string // low, medium, high, critical
	Description string
	Score       int
}

// SQLi patterns
var sqliPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\bunion\b.+\bselect\b|\bselect\b.+\bfrom\b)`),
	regexp.MustCompile(`(?i)(\bdrop\b.+\btable\b|\bdelete\b.+\bfrom\b)`),
	regexp.MustCompile(`(?i)(\'|\"|;|--|\bor\b\s+[\d\'\"]=[\d\'\"]|\band\b\s+[\d\'\"]=[\d\'\"])`),
	regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\s*\(|waitfor\s+delay)`),
	regexp.MustCompile(`(?i)(information_schema|sysobjects|syscolumns)`),
}

// XSS patterns
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[\s>]`),
	regexp.MustCompile(`(?i)(javascript:|vbscript:|onload=|onerror=|onclick=)`),
	regexp.MustCompile(`(?i)(<iframe|<object|<embed|<svg)`),
	regexp.MustCompile(`(?i)(alert\s*\(|confirm\s*\(|prompt\s*\()`),
	regexp.MustCompile(`(?i)(document\.cookie|window\.location)`),
}

// Path traversal patterns
var pathTraversalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(\.\.\/|\.\.\\)`),
	regexp.MustCompile(`(?i)(\/etc\/passwd|\/etc\/shadow|\/proc\/self)`),
	regexp.MustCompile(`(?i)(\.\.%2f|\.\.%5c|%2e%2e)`),
}

// Scanner/bot user agents
var scannerAgents = []string{
	"nikto", "sqlmap", "nmap", "masscan", "zgrab",
	"nuclei", "dirbuster", "gobuster", "wfuzz", "burpsuite",
	"python-requests", "go-http-client", "libwww-perl",
	"wp-scan", "acunetix", "nessus", "openvas",
}

// Sensitive paths that shouldn't be accessed
var sensitivePaths = []string{
	"/admin", "/wp-admin", "/phpmyadmin", "/.env",
	"/config", "/.git", "/backup", "/db",
	"/shell", "/cmd", "/exec", "/cgi-bin",
}

func AnalyzeRequest(path, userAgent, ip string, statusCode int) []Threat {
	var threats []Threat
	combined := strings.ToLower(path + " " + userAgent)

	// SQLi check
	for _, pattern := range sqliPatterns {
		if pattern.MatchString(combined) {
			threats = append(threats, Threat{
				Type:        ThreatSQLi,
				Severity:    "critical",
				Description: "SQL injection attempt detected in request",
				Score:       90,
			})
			break
		}
	}

	// XSS check
	for _, pattern := range xssPatterns {
		if pattern.MatchString(combined) {
			threats = append(threats, Threat{
				Type:        ThreatXSS,
				Severity:    "high",
				Description: "XSS attempt detected in request",
				Score:       75,
			})
			break
		}
	}

	// Path traversal
	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(path) {
			threats = append(threats, Threat{
				Type:        ThreatPathTraversal,
				Severity:    "high",
				Description: "Path traversal attempt detected",
				Score:       80,
			})
			break
		}
	}

	// Scanner bot detection
	uaLower := strings.ToLower(userAgent)
	for _, agent := range scannerAgents {
		if strings.Contains(uaLower, agent) {
			threats = append(threats, Threat{
				Type:        ThreatScannerBot,
				Severity:    "medium",
				Description: "Known scanner/bot detected: " + agent,
				Score:       60,
			})
			break
		}
	}

	return threats
}

func ThreatScore(threats []Threat) int {
	score := 0
	for _, t := range threats {
		score += t.Score
	}
	if score > 100 {
		score = 100
	}
	return score
}
