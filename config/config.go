package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	JWT      JWTConfig
	Firebase FirebaseConfig
	CORS     CORSConfig
	RateLimit RateLimitConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port        string
	Host        string
	Environment string
}

type JWTConfig struct {
	Secret                string
	Expiration            time.Duration
	RefreshTokenExpiration time.Duration
}

type FirebaseConfig struct {
	ProjectID       string
	CredentialsPath string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Host:        getEnv("HOST", "0.0.0.0"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		JWT: JWTConfig{
			Secret:                getEnv("JWT_SECRET", "dev-secret-key"),
			Expiration:            parseDuration(getEnv("JWT_EXPIRATION", "30m"), 30*time.Minute),
			RefreshTokenExpiration: parseDuration(getEnv("REFRESH_TOKEN_EXPIRATION", "7d"), 7*24*time.Hour),
		},
		Firebase: FirebaseConfig{
			ProjectID:       getEnv("FIREBASE_PROJECT_ID", "gatekeeper-e1209"),
			CredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", "./serviceAccountKey.json"),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseStringSlice(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		},
		RateLimit: RateLimitConfig{
			Requests: parseInt(getEnv("RATE_LIMIT_REQUESTS", "100"), 100),
			Window:   parseDuration(getEnv("RATE_LIMIT_WINDOW", "60"), 60*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	// Handle simple formats like "30m", "7d", "60"
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	// If it's just a number, assume seconds
	if i, err := strconv.Atoi(s); err == nil {
		return time.Duration(i) * time.Second
	}
	return defaultValue
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	for i := 0; i < len(s); {
		end := i
		for end < len(s) && s[end] != ',' {
			end++
		}
		if i < end {
			result = append(result, s[i:end])
		}
		i = end + 1
	}
	return result
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) Validate() {
	if c.JWT.Secret == "dev-secret-key" && c.IsProduction() {
		log.Fatal("JWT_SECRET must be set in production")
	}
	if c.Firebase.ProjectID == "" {
		log.Fatal("FIREBASE_PROJECT_ID must be set")
	}
	if _, err := os.Stat(c.Firebase.CredentialsPath); os.IsNotExist(err) {
		log.Fatalf("Firebase credentials file not found: %s", c.Firebase.CredentialsPath)
	}
}
