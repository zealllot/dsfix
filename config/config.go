package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = ".dsfix.yaml"
)

// Config represents the dsfix configuration
type Config struct {
	// DeepSource configuration
	DeepSource DeepSourceConfig `yaml:"deepsource"`

	// Repository configuration
	Repository RepositoryConfig `yaml:"repository"`

	// Filter configuration
	Filter FilterConfig `yaml:"filter"`
}

// DeepSourceConfig contains DeepSource API settings
type DeepSourceConfig struct {
	APIToken string `yaml:"api_token"` // Can also be set via DEEPSOURCE_API_TOKEN env var
}

// RepositoryConfig contains repository settings
type RepositoryConfig struct {
	Owner string `yaml:"owner"` // GitHub owner/organization
	Name  string `yaml:"name"`  // Repository name
	Path  string `yaml:"path"`  // Local path to the repository
}

// FilterConfig contains issue filter settings
type FilterConfig struct {
	Categories []string `yaml:"categories"` // Bug Risk, Anti-pattern, Security, etc.
	Severities []string `yaml:"severities"` // critical, major, minor
	Limit      int      `yaml:"limit"`      // Maximum number of issues to fetch
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	if token := os.Getenv("DEEPSOURCE_API_TOKEN"); token != "" {
		cfg.DeepSource.APIToken = token
	}

	return &cfg, nil
}

// LoadFromDir loads configuration from a directory
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, DefaultConfigFile)
	return Load(configPath)
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DeepSource.APIToken == "" {
		return fmt.Errorf("DeepSource API token is required (set in config or DEEPSOURCE_API_TOKEN env var)")
	}

	if c.Repository.Owner == "" {
		return fmt.Errorf("repository owner is required")
	}

	if c.Repository.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DeepSource: DeepSourceConfig{},
		Repository: RepositoryConfig{},
		Filter:     FilterConfig{},
	}
}

// GenerateTemplate generates a configuration template
func GenerateTemplate() string {
	return `# DSFix Configuration
# DeepSource + Windsurf Integration

deepsource:
  # API token for DeepSource (or set DEEPSOURCE_API_TOKEN env var)
  api_token: ""

repository:
  # GitHub owner/organization
  owner: "theplant"
  # Repository name
  name: "mcd-website"
  # Local path to the repository (leave empty for current directory)
  path: ""

filter:
  # Categories to include (leave empty for all)
  # Options: Bug Risk, Anti-pattern, Security, Performance, Typecheck, Style, Documentation
  categories: []
  
  # Severities to include (leave empty for all)
  # Options: critical, major, minor
  severities: []
  
  # Maximum number of issues to fetch (leave empty for unlimited)
  limit:
`
}
