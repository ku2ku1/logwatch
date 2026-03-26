package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logs     LogsConfig     `yaml:"logs"`
	GeoIP    GeoIPConfig    `yaml:"geoip"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LogsConfig struct {
	NginxAccess string `yaml:"nginx_access"`
	NginxError  string `yaml:"nginx_error"`
	AuthLog     string `yaml:"auth_log"`
	Fail2banLog string `yaml:"fail2ban_log"`
	UFWLog      string `yaml:"ufw_log"`
}

type GeoIPConfig struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 8080
	cfg.Database.Path = "./data/logvance.db"
	cfg.Logs.NginxAccess = "/var/log/nginx/access.log"
	cfg.Logs.AuthLog = "/var/log/auth.log"
	cfg.Logs.Fail2banLog = "/var/log/fail2ban.log"
	cfg.Logs.UFWLog = "/var/log/ufw.log"
	cfg.GeoIP.Path = "data/geoip/GeoLite2-City.mmdb"

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
