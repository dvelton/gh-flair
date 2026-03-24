package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/dvelton/gh-flair/internal/model"
	"gopkg.in/yaml.v3"
)

// RepoConfig is a single tracked repository entry in the config file.
type RepoConfig struct {
	Name     string            `yaml:"name"`
	Packages map[string]string `yaml:"packages,omitempty"`
}

// MilestoneThresholds holds per-metric threshold progressions.
type MilestoneThresholds struct {
	Stars        []int `yaml:"stars"`
	Forks        []int `yaml:"forks"`
	Contributors []int `yaml:"contributors"`
	Downloads    []int `yaml:"downloads"`
}

// Settings holds global behavioural settings.
type Settings struct {
	Streaks             bool                `yaml:"streaks"`
	NotableThreshold    int                 `yaml:"notable_threshold"`
	Quiet               bool                `yaml:"quiet"`
	MilestoneThresholds MilestoneThresholds `yaml:"milestone_thresholds"`
}

// Config is the top-level config file structure.
type Config struct {
	Repos    []RepoConfig `yaml:"repos"`
	Settings Settings     `yaml:"settings"`
}

// ConfigPath returns the default config file path.
func ConfigPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

// DBPath returns the default database path.
func DBPath() string {
	return filepath.Join(configDir(), "flair.db")
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Repos: []RepoConfig{},
		Settings: Settings{
			Streaks:          true,
			NotableThreshold: 1000,
			Quiet:            false,
			MilestoneThresholds: MilestoneThresholds{
				Stars:        model.StarThresholds,
				Forks:        model.ForkThresholds,
				Contributors: model.ContributorThresholds,
				Downloads:    model.DownloadThresholds,
			},
		},
	}
}

// Load reads the config from the default path. If the file does not exist it
// returns DefaultConfig rather than an error.
func Load() (*Config, error) {
	return LoadFrom(ConfigPath())
}

// LoadFrom reads the config from path. If the file does not exist it returns
// DefaultConfig rather than an error.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes cfg to the default config path, creating directories as needed.
func Save(cfg *Config) error {
	return SaveTo(cfg, ConfigPath())
}

// SaveTo writes cfg to path, creating parent directories as needed.
func SaveTo(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// configDir returns ~/.config/gh-flair.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-flair")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "gh-flair")
	}
	return filepath.Join(home, ".config", "gh-flair")
}
