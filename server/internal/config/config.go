package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr   string `yaml:"addr"`
		Secret string `yaml:"secret"`
	} `yaml:"server"`
	Domain string `yaml:"domain"`
	DB     struct {
		Driver string `yaml:"driver"` // sqlite | mysql | postgres
		DSN    string `yaml:"dsn"`
	} `yaml:"db"`
	SMTPIn struct {
		Enabled bool   `yaml:"enabled"`
		Addr    string `yaml:"addr"`
		MaxMB   int    `yaml:"max_mb"`
	} `yaml:"smtp_in"`
	Mail struct {
		Mode  string `yaml:"mode"` // relay | direct
		Relay struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			TLS      string `yaml:"tls"` // ssl | starttls | none
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"relay"`
		HeloDomain string `yaml:"helo_domain"`
	} `yaml:"mail"`
	FetchIntervalSec int    `yaml:"fetch_interval_sec"`
	AESKey           string `yaml:"aes_key"`
	WebDist          string `yaml:"web_dist"` // 前端dist目录，默认../web/dist（本地开发布局）
}

func Load(path string) (*Config, error) {
	c := &Config{}
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(b) > 0 {
		if err := yaml.Unmarshal(b, c); err != nil {
			return nil, err
		}
	}
	c.applyDefaults()
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":38083"
	}
	if c.Server.Secret == "" {
		c.Server.Secret = "caoyou-default-secret-change-me"
	}
	if c.Domain == "" {
		c.Domain = "localhost"
	}
	if c.DB.Driver == "" {
		c.DB.Driver = "sqlite"
	}
	if c.DB.DSN == "" {
		c.DB.DSN = "caoyou.db"
	}
	if c.SMTPIn.Addr == "" {
		c.SMTPIn.Addr = ":25"
	}
	if c.SMTPIn.MaxMB <= 0 {
		c.SMTPIn.MaxMB = 32
	}
	if c.Mail.Mode == "" {
		c.Mail.Mode = "relay"
	}
	if c.Mail.Relay.TLS == "" {
		c.Mail.Relay.TLS = "ssl"
	}
	if c.Mail.HeloDomain == "" {
		c.Mail.HeloDomain = c.Domain
	}
	if c.FetchIntervalSec <= 0 {
		c.FetchIntervalSec = 300
	}
	if c.AESKey == "" {
		c.AESKey = "caoyou-default-aes-key-change-me"
	}
	if c.WebDist == "" {
		c.WebDist = "../web/dist"
	}
}
