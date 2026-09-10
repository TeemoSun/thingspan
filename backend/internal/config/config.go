package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppPassword              string
	JWTSecret                string
	AccessTokenExpireMinutes int
	RefreshTokenExpireDays   int

	TZ      string
	DataDir string
	Port    int

	StaticDir string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	MailTo       string

	ReminderLeadDays  string
	ReminderCheckHour int

	LeadDays []int
	Location *time.Location
}

// LoadEnvFiles reads key-value pairs from .env files if present (does not overwrite existing environment variables)
func LoadEnvFiles(paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// remove quotes if present
			if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
				(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
				v = v[1 : len(v)-1]
			}
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
		_ = f.Close()
	}
}

func getEnv(key, def string) string {
	if v, exists := os.LookupEnv(key); exists && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, exists := os.LookupEnv(key); exists && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func LoadConfig() (*Config, error) {
	// Try loading from possible locations: .env, ../.env, /data/.env
	dataDir := os.Getenv("DATA_DIR")
	var paths []string
	if dataDir != "" {
		paths = append(paths, filepath.Join(dataDir, ".env"))
	}
	paths = append(paths, ".env", "../.env")
	LoadEnvFiles(paths...)

	cfg := &Config{
		AppPassword:              getEnv("APP_PASSWORD", ""),
		JWTSecret:                getEnv("JWT_SECRET", ""),
		AccessTokenExpireMinutes: getEnvInt("ACCESS_TOKEN_EXPIRE_MINUTES", 30),
		RefreshTokenExpireDays:   getEnvInt("REFRESH_TOKEN_EXPIRE_DAYS", 30),
		TZ:                       getEnv("TZ", "Asia/Shanghai"),
		DataDir:                  getEnv("DATA_DIR", "./data"),
		Port:                     getEnvInt("PORT", 8000),
		StaticDir:                getEnv("STATIC_DIR", "./static"),
		SMTPHost:                 getEnv("SMTP_HOST", ""),
		SMTPPort:                 getEnvInt("SMTP_PORT", 465),
		SMTPUser:                 getEnv("SMTP_USER", ""),
		SMTPPassword:             getEnv("SMTP_PASSWORD", ""),
		MailTo:                   getEnv("MAIL_TO", ""),
		ReminderLeadDays:         getEnv("REMINDER_LEAD_DAYS", "30,7,1"),
		ReminderCheckHour:        getEnvInt("REMINDER_CHECK_HOUR", 9),
	}

	// Parse lead days
	var days []int
	for _, part := range strings.Split(cfg.ReminderLeadDays, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil && n >= 0 {
			days = append(days, n)
		}
	}
	sort.Ints(days)
	cfg.LeadDays = days

	// Load timezone location
	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	cfg.Location = loc

	return cfg, nil
}

func (c *Config) ValidateSecrets() error {
	if c.AppPassword == "" || c.AppPassword == "admin" || c.AppPassword == "change-me" {
		return fmt.Errorf("APP_PASSWORD 未配置或仍为默认值，请设置 .env 中的 APP_PASSWORD 后重启")
	}
	if c.JWTSecret == "" || c.JWTSecret == "change-me" {
		return fmt.Errorf("JWT_SECRET 未配置或仍为默认值，请用 `openssl rand -hex 32` 生成并写入 .env 后重启")
	}
	return nil
}

func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "thingspan.db")
}

func (c *Config) LocalNow() time.Time {
	return time.Now().In(c.Location)
}

func (c *Config) TodayStr() string {
	return c.LocalNow().Format("2006-01-02")
}

