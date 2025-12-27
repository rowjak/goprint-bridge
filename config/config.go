package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	SelectedPrinter string `mapstructure:"selected_printer" json:"selected_printer"`
	Port            int    `mapstructure:"port" json:"port"`
	AutoStart       bool   `mapstructure:"auto_start" json:"auto_start"`
}

var cfg *Config

// LoadConfig loads configuration from config.yaml
func LoadConfig() (*Config, error) {
	// Get executable path for config location
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	configPath := filepath.Dir(execPath)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("selected_printer", "")
	viper.SetDefault("port", 9999)
	viper.SetDefault("auto_start", false)

	// Try to read existing config
	if err := viper.ReadInConfig(); err != nil {
		// Config doesn't exist, create with defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cfg = &Config{
				SelectedPrinter: "",
				Port:            9999,
				AutoStart:       false,
			}
			// Save default config
			return cfg, SaveConfig(cfg)
		}
		return nil, err
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig saves the configuration to config.yaml
func SaveConfig(c *Config) error {
	viper.Set("selected_printer", c.SelectedPrinter)
	viper.Set("port", c.Port)
	viper.Set("auto_start", c.AutoStart)

	// Ensure config file exists
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = "config.yaml"
	}

	return viper.WriteConfigAs(configFile)
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	if cfg == nil {
		cfg = &Config{
			SelectedPrinter: "",
			Port:            9999,
			AutoStart:       false,
		}
	}
	return cfg
}

// UpdateConfig updates and saves the configuration
func UpdateConfig(printer string, port int, autoStart bool) error {
	cfg = &Config{
		SelectedPrinter: printer,
		Port:            port,
		AutoStart:       autoStart,
	}
	return SaveConfig(cfg)
}
