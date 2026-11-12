package config

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDB     string
	JWTSecret   string
	JWTTTLHours int
	CORSOrigins []string
	RecsMode    string
	CacheTTL    time.Duration
}

func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:     getEnv("MONGO_DB", "braille_library"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		CORSOrigins: splitList(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")),
	}
	cfg.RecsMode = getEnv("RECS_MODE", "precomputed")
	if cfg.RecsMode != "precomputed" && cfg.RecsMode != "live" {
		return cfg, errors.New("RECS_MODE must be precomputed or live")
	}
	cacheTTL, err := time.ParseDuration(getEnv("CACHE_TTL", "60s"))
	if err != nil || cacheTTL < 0 {
		return cfg, errors.New("CACHE_TTL must be a duration like 60s, or 0 to turn the cache off")
	}
	cfg.CacheTTL = cacheTTL
	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET is required")
	}

	ttl, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "24"))
	if err != nil || ttl <= 0 {
		return cfg, errors.New("JWT_TTL_HOURS must be a positive integer")
	}
	cfg.JWTTTLHours = ttl
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv reads KEY=VALUE lines from a file into the environment.
// Variables that are already set are not overwritten.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// splitList splits a comma-separated value and drops empty parts.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
