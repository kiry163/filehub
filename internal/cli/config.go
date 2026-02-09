package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Endpoint       string `yaml:"endpoint"`
	LocalKey       string `yaml:"local_key"`
	PublicEndpoint string `yaml:"public_endpoint"`
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

func InitConfig(endpoint, localKey, publicEndpoint string) (string, error) {
	if endpoint == "" {
		endpoint = prompt("API endpoint", "http://localhost:8080")
	}
	if localKey == "" {
		localKey = prompt("Local key", "")
	}
	if publicEndpoint == "" {
		publicEndpoint = prompt("Public endpoint (optional)", "")
	}
	if publicEndpoint == "" {
		resolved, err := resolvePublicEndpoint(endpoint)
		if err != nil {
			return "", err
		}
		publicEndpoint = resolved
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
	data, err := yaml.Marshal(Config{Endpoint: endpoint, LocalKey: localKey, PublicEndpoint: publicEndpoint})
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

func publicBaseURL(cfg Config) string {
	if cfg.PublicEndpoint != "" {
		return strings.TrimRight(cfg.PublicEndpoint, "/")
	}
	return strings.TrimRight(cfg.Endpoint, "/")
}

func buildPublicURL(cfg Config, path string) string {
	return publicBaseURL(cfg) + path
}

func resolvePublicEndpoint(endpoint string) (string, error) {
	publicIP, err := fetchPublicIP()
	if err != nil {
		return "", err
	}
	port := endpointPort(endpoint)
	return fmt.Sprintf("http://%s:%s", publicIP, port), nil
}

func fetchPublicIP() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ip.sb/jsonip")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ip service failed: %s", resp.Status)
	}
	var payload struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.IP == "" {
		return "", errors.New("ip service returned empty ip")
	}
	return payload.IP, nil
}

func endpointPort(endpoint string) string {
	parsed, err := parseEndpoint(endpoint)
	if err != nil {
		return "80"
	}
	if port := parsed.Port(); port != "" {
		return port
	}
	return "80"
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
