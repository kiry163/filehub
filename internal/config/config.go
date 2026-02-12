package config

import (
	"errors"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Upload   UploadConfig   `yaml:"upload"`
	Minio    MinioConfig    `yaml:"minio"`
}

type ServerConfig struct {
	Port           int    `yaml:"port"`
	LogLevel       string `yaml:"log_level"`
	PublicEndpoint string `yaml:"public_endpoint"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	JWTSecret         string `yaml:"jwt_secret"`
	JWTExpireHours    int64  `yaml:"jwt_expire_hours"`
	RefreshExpireDays int64  `yaml:"refresh_expire_days"`
	AdminUsername     string `yaml:"admin_username"`
	AdminPassword     string `yaml:"admin_password"`
	LocalKey          string `yaml:"local_key"`
}

type UploadConfig struct {
	MaxSizeMB int64 `yaml:"max_size_mb"`
}

type MinioConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
	Region    string `yaml:"region"`
}

func Load(path string) (Config, error) {
	config := defaultConfig()
	if path == "" {
		path = "config.yaml"
	}
	if err := loadConfigFile(path, &config); err != nil {
		return Config{}, err
	}
	overrideWithEnv(&config)
	if config.Auth.JWTSecret == "" {
		return Config{}, errors.New("missing auth.jwt_secret")
	}
	if config.Auth.AdminPassword == "" {
		return Config{}, errors.New("missing auth.admin_password")
	}
	if config.Minio.Endpoint == "" || config.Minio.AccessKey == "" || config.Minio.SecretKey == "" {
		return Config{}, errors.New("missing minio config")
	}
	if config.Minio.Bucket == "" {
		config.Minio.Bucket = "filehub"
	}
	return config, nil
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Database: DatabaseConfig{
			Path: "./data/filehub.db",
		},
		Auth: AuthConfig{
			JWTExpireHours:    24,
			RefreshExpireDays: 7,
			AdminUsername:     "admin",
		},
		Upload: UploadConfig{
			MaxSizeMB: 1024,
		},
		Minio: MinioConfig{
			Bucket: "filehub",
			UseSSL: false,
		},
	}
}

func loadConfigFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return yaml.Unmarshal(data, config)
}

func overrideWithEnv(config *Config) {
	if value := os.Getenv("FILEHUB_SERVER_PORT"); value != "" {
		config.Server.Port = parseInt(value, config.Server.Port)
	}
	if value := os.Getenv("FILEHUB_SERVER_LOG_LEVEL"); value != "" {
		config.Server.LogLevel = value
	}
	if value := os.Getenv("FILEHUB_SERVER_PUBLIC_ENDPOINT"); value != "" {
		config.Server.PublicEndpoint = value
	}
	if value := os.Getenv("FILEHUB_DATABASE_PATH"); value != "" {
		config.Database.Path = value
	}
	if value := os.Getenv("FILEHUB_AUTH_JWT_SECRET"); value != "" {
		config.Auth.JWTSecret = value
	}
	if value := os.Getenv("FILEHUB_AUTH_JWT_EXPIRE_HOURS"); value != "" {
		config.Auth.JWTExpireHours = parseInt64(value, config.Auth.JWTExpireHours)
	}
	if value := os.Getenv("FILEHUB_AUTH_REFRESH_EXPIRE_DAYS"); value != "" {
		config.Auth.RefreshExpireDays = parseInt64(value, config.Auth.RefreshExpireDays)
	}
	if value := os.Getenv("FILEHUB_AUTH_ADMIN_USERNAME"); value != "" {
		config.Auth.AdminUsername = value
	}
	if value := os.Getenv("FILEHUB_AUTH_ADMIN_PASSWORD"); value != "" {
		config.Auth.AdminPassword = value
	}
	if value := os.Getenv("FILEHUB_AUTH_LOCAL_KEY"); value != "" {
		config.Auth.LocalKey = value
	}
	if value := os.Getenv("FILEHUB_UPLOAD_MAX_SIZE_MB"); value != "" {
		config.Upload.MaxSizeMB = parseInt64(value, config.Upload.MaxSizeMB)
	}
	if value := os.Getenv("FILEHUB_MINIO_ENDPOINT"); value != "" {
		config.Minio.Endpoint = value
	}
	if value := os.Getenv("FILEHUB_MINIO_ACCESS_KEY"); value != "" {
		config.Minio.AccessKey = value
	}
	if value := os.Getenv("FILEHUB_MINIO_SECRET_KEY"); value != "" {
		config.Minio.SecretKey = value
	}
	if value := os.Getenv("FILEHUB_MINIO_BUCKET"); value != "" {
		config.Minio.Bucket = value
	}
	if value := os.Getenv("FILEHUB_MINIO_USE_SSL"); value != "" {
		config.Minio.UseSSL = parseBool(value, config.Minio.UseSSL)
	}
	if value := os.Getenv("FILEHUB_MINIO_REGION"); value != "" {
		config.Minio.Region = value
	}
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64(value string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
