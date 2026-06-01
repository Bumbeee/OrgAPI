package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddress string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration

	DatabaseDSN       string
	DBMaxConns        int
	DBMinConns        int
	DBConnMaxIdleTime time.Duration
	MigrationsDir     string

	LogLevel string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", getEnv("APP_PORT", ":8080")),
		DatabaseDSN:   databaseDSN(),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),
		LogLevel:      getEnv("LOG_LEVEL", "debug"),
	}

	var err error
	cfg.ReadTimeout, err = parseDuration("READ_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}
	cfg.WriteTimeout, err = parseDuration("WRITE_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}
	cfg.DBMaxConns, err = getIntEnv("DB_MAX_CONNS", "10")
	if err != nil {
		return nil, err
	}
	cfg.DBMinConns, err = getIntEnv("DB_MIN_CONNS", "2")
	if err != nil {
		return nil, err
	}
	cfg.DBConnMaxIdleTime, err = parseDuration("DB_CONN_MAX_IDLE_TIME", "30m")
	if err != nil {
		return nil, err
	}

	return cfg, cfg.validate()
}

func databaseDSN() string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}

	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	name := os.Getenv("DB_NAME")
	if host == "" || user == "" || name == "" {
		return ""
	}

	port := getEnv("DB_PORT", "5432")
	password := os.Getenv("DB_PASSWORD")
	sslMode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host,
		user,
		password,
		name,
		port,
		sslMode,
	)
}

func (c *Config) validate() error {
	if c.DatabaseDSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getIntEnv(key, defaultVal string) (int, error) {
	val := getEnv(key, defaultVal)
	d, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", key, val, err)
	}
	return d, nil
}

func parseDuration(key, defaultVal string) (time.Duration, error) {
	val := getEnv(key, defaultVal)
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s duration %q: %w", key, val, err)
	}
	return d, nil
}
