package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	APIURL    string `yaml:"api-url"`
	APIKey    string `yaml:"api-key"`
	AccountID string `yaml:"account-id,omitempty"`
	ProjectID string `yaml:"project-id,omitempty"`
}

type Config struct {
	CurrentProfile string             `yaml:"current-profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".raff")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	cfg := &Config{
		CurrentProfile: "default",
		Profiles:       make(map[string]Profile),
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func (c *Config) ActiveProfile() Profile {
	if p, ok := c.Profiles[c.CurrentProfile]; ok {
		return p
	}
	return Profile{}
}

// Resolve returns the effective value for a setting using priority:
// CLI flag > env var > config profile
func Resolve(flagVal, envKey string, configVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return configVal
}

// PromptInput reads a line from stdin with an optional default value shown in brackets.
func PromptInput(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	if line == "" {
		return defaultVal
	}
	return line
}
