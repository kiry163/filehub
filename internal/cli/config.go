package cli

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Endpoint string `yaml:"endpoint"`
	LocalKey string `yaml:"local_key"`
}

func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Endpoint == "" || cfg.LocalKey == "" {
		return Config{}, errors.New("invalid config: endpoint/local_key required")
	}
	return cfg, nil
}

func InitConfig(endpoint, localKey string) (string, error) {
	if endpoint == "" {
		endpoint = prompt("API endpoint", "http://localhost:8080")
	}
	if localKey == "" {
		localKey = prompt("Local key", "")
	}
	if endpoint == "" || localKey == "" {
		return "", errors.New("endpoint and local_key are required")
	}
	path, err := configPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(Config{Endpoint: endpoint, LocalKey: localKey})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "filehub-cli", "config.yaml"), nil
}

func prompt(label, fallback string) string {
	reader := bufio.NewReader(os.Stdin)
	if fallback != "" {
		fmt.Printf("%s [%s]: ", label, fallback)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return text
}

func parseEndpoint(endpoint string) (*url.URL, error) {
	if endpoint == "" {
		return nil, errors.New("endpoint required")
	}
	value := endpoint
	if !strings.Contains(endpoint, "://") {
		value = "http://" + endpoint
	}
	return url.Parse(value)
}
