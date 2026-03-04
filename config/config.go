package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a YAML config file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type Config struct {
	Title      string     `yaml:"title"`
	Categories []Category `yaml:"categories"`
	Weather    *Weather   `yaml:"weather,omitempty"`
}

type Category struct {
	Name         string `yaml:"name"`
	Enabled      *bool  `yaml:"enabled,omitempty"`
	Private      *bool  `yaml:"private,omitempty"`
	OpenInNewTab *bool  `yaml:"open_in_new_tab,omitempty"`
	Links        []Link `yaml:"links"`
}

type Link struct {
	Name         string   `yaml:"name"`
	URL          string   `yaml:"url"`
	MobileURL    string   `yaml:"mobile_url,omitempty"`
	Icon         string   `yaml:"icon,omitempty"`
	Enabled      *bool    `yaml:"enabled,omitempty"`
	OpenInNewTab *bool    `yaml:"open_in_new_tab,omitempty"`
	Private      *bool    `yaml:"private,omitempty"`
	Keywords     []string `yaml:"keywords,omitempty"`
}

type Weather struct {
	Latitude     float64 `yaml:"default_latitude"`
	Longitude    float64 `yaml:"default_longitude"`
	Unit         string  `yaml:"unit"`
	CacheMinutes int     `yaml:"cache_minutes"`
}
