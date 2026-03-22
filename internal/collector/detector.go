package collector

import (
	"os"
	"os/exec"
	"strings"
)

type ServiceType string

const (
	ServiceNginx    ServiceType = "nginx"
	ServiceApache   ServiceType = "apache"
	ServiceFail2ban ServiceType = "fail2ban"
	ServiceUFW      ServiceType = "ufw"
	ServiceDocker   ServiceType = "docker"
	ServiceSSH      ServiceType = "ssh"
	ServiceSystemd  ServiceType = "systemd"
	ServiceClamAV   ServiceType = "clamav"
	ServiceOpenBao  ServiceType = "openbao"
	ServicePostfix  ServiceType = "postfix"
)

type DetectedService struct {
	Type        ServiceType
	Name        string
	LogPaths    []string
	Running     bool
	Version     string
	Description string
}

// AutoDetect — VPS pe jo bhi services hain sab dhundho
func AutoDetect() []DetectedService {
	var services []DetectedService

	checks := []func() *DetectedService{
		detectNginx,
		detectApache,
		detectFail2ban,
		detectUFW,
		detectDocker,
		detectSSH,
		detectClamAV,
		detectOpenBao,
		detectPostfix,
	}

	for _, check := range checks {
		if svc := check(); svc != nil {
			services = append(services, *svc)
		}
	}

	// Systemd services — saare running services
	systemdSvcs := detectSystemdServices()
	services = append(services, systemdSvcs...)

	return services
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isRunning(name string) bool {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

func getVersion(cmd string, args ...string) string {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func detectNginx() *DetectedService {
	paths := []string{
		"/var/log/nginx/access.log",
		"/var/log/nginx/error.log",
	}
	var found []string
	for _, p := range paths {
		if fileExists(p) {
			found = append(found, p)
		}
	}
	if len(found) == 0 && !isRunning("nginx") {
		return nil
	}
	return &DetectedService{
		Type:        ServiceNginx,
		Name:        "Nginx",
		LogPaths:    found,
		Running:     isRunning("nginx"),
		Version:     getVersion("nginx", "-v"),
		Description: "Web server — access + error logs",
	}
}

func detectApache() *DetectedService {
	paths := []string{
		"/var/log/apache2/access.log",
		"/var/log/apache2/error.log",
		"/var/log/httpd/access_log",
	}
	var found []string
	for _, p := range paths {
		if fileExists(p) {
			found = append(found, p)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &DetectedService{
		Type:        ServiceApache,
		Name:        "Apache2",
		LogPaths:    found,
		Running:     isRunning("apache2") || isRunning("httpd"),
		Description: "Web server — access + error logs",
	}
}

func detectFail2ban() *DetectedService {
	if !fileExists("/var/log/fail2ban.log") && !isRunning("fail2ban") {
		return nil
	}
	return &DetectedService{
		Type:        ServiceFail2ban,
		Name:        "Fail2ban",
		LogPaths:    []string{"/var/log/fail2ban.log"},
		Running:     isRunning("fail2ban"),
		Version:     getVersion("fail2ban-client", "--version"),
		Description: "Intrusion prevention — bans, jails",
	}
}

func detectUFW() *DetectedService {
	if !fileExists("/var/log/ufw.log") && !fileExists("/usr/sbin/ufw") {
		return nil
	}
	return &DetectedService{
		Type:        ServiceUFW,
		Name:        "UFW Firewall",
		LogPaths:    []string{"/var/log/ufw.log"},
		Running:     true,
		Description: "Firewall — blocked connections, rules",
	}
}

func detectDocker() *DetectedService {
	_, err := exec.LookPath("docker")
	if err != nil {
		return nil
	}
	return &DetectedService{
		Type:        ServiceDocker,
		Name:        "Docker",
		LogPaths:    []string{},
		Running:     isRunning("docker"),
		Version:     getVersion("docker", "--version"),
		Description: "Container runtime — container logs",
	}
}

func detectSSH() *DetectedService {
	paths := []string{"/var/log/auth.log", "/var/log/secure"}
	var found []string
	for _, p := range paths {
		if fileExists(p) {
			found = append(found, p)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &DetectedService{
		Type:        ServiceSSH,
		Name:        "SSH / Auth",
		LogPaths:    found,
		Running:     isRunning("sshd") || isRunning("ssh"),
		Description: "Login attempts, sudo, auth failures",
	}
}

func detectClamAV() *DetectedService {
	_, err := exec.LookPath("clamscan")
	if err != nil && !isRunning("clamav-daemon") {
		return nil
	}
	paths := []string{
		"/var/log/clamav/clamav.log",
		"/var/log/clamav/freshclam.log",
	}
	var found []string
	for _, p := range paths {
		if fileExists(p) {
			found = append(found, p)
		}
	}
	return &DetectedService{
		Type:        ServiceClamAV,
		Name:        "ClamAV",
		LogPaths:    found,
		Running:     isRunning("clamav-daemon"),
		Description: "Antivirus — scans, detections, quarantine",
	}
}

func detectOpenBao() *DetectedService {
	_, err := exec.LookPath("bao")
	if err != nil {
		return nil
	}
	return &DetectedService{
		Type:        ServiceOpenBao,
		Name:        "OpenBao / Vault",
		LogPaths:    []string{},
		Running:     isRunning("openbao"),
		Description: "Secret management — audit logs",
	}
}

func detectPostfix() *DetectedService {
	if !fileExists("/var/log/mail.log") && !isRunning("postfix") {
		return nil
	}
	return &DetectedService{
		Type:        ServicePostfix,
		Name:        "Postfix (Mail)",
		LogPaths:    []string{"/var/log/mail.log"},
		Running:     isRunning("postfix"),
		Description: "Mail server — delivery, errors",
	}
}

func detectSystemdServices() []DetectedService {
	out, err := exec.Command("systemctl", "list-units", "--type=service",
		"--state=active", "--no-pager", "--no-legend").Output()
	if err != nil {
		return nil
	}

	// Known services jo already handle hote hain
	skip := map[string]bool{
		"nginx": true, "apache2": true, "fail2ban": true,
		"docker": true, "sshd": true, "clamav-daemon": true,
		"openbao": true, "postfix": true,
	}

	var services []DetectedService
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ".service")
		if skip[name] {
			continue
		}
		// Custom user services (n8n, etc)
		if len(fields) >= 4 {
			desc := strings.Join(fields[4:], " ")
			services = append(services, DetectedService{
				Type:        ServiceSystemd,
				Name:        name,
				Running:     true,
				Description: desc,
			})
		}
	}
	return services
}
