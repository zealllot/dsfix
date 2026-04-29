package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = ".dsfix.yaml"

// Config represents the dsfix configuration.
type Config struct {
	DeepSource DeepSourceConfig `yaml:"deepsource"`
	Repository RepositoryConfig `yaml:"repository"`
	Filter     FilterConfig     `yaml:"filter"`
	Verify     VerifyConfig     `yaml:"verify"`
}

// DeepSourceConfig contains DeepSource API settings.
type DeepSourceConfig struct {
	APIToken string `yaml:"api_token"` // Can also be set via DEEPSOURCE_API_TOKEN env var
}

// RepositoryConfig contains repository settings.
type RepositoryConfig struct {
	Owner string `yaml:"owner"` // VCS owner/organization
	Name  string `yaml:"name"`  // Repository name
}

// FilterConfig contains issue filter settings.
type FilterConfig struct {
	Categories   []string `yaml:"categories"`              // Bug Risk, Anti-pattern, Security, etc.
	Severities   []string `yaml:"severities"`              // critical, major, minor
	Limit        int      `yaml:"limit"`                   // Maximum number of issues to fetch
	PathsInclude []string `yaml:"paths_include,omitempty"` // glob patterns; if non-empty, only matching paths are kept
	PathsExclude []string `yaml:"paths_exclude,omitempty"` // glob patterns; matching paths are dropped
}

// VerifyConfig contains the post-fix verification command.
type VerifyConfig struct {
	Command string `yaml:"command"` // e.g. "go build ./..." or "pnpm typecheck"; empty means let the AI choose
}

// Load reads configuration from a file. Token is overridden by DEEPSOURCE_API_TOKEN env var.
// If a token is found in the YAML, a one-line warning is written to stderr.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.DeepSource.APIToken != "" && os.Getenv("DEEPSOURCE_API_TOKEN") == "" {
		fmt.Fprintf(os.Stderr, "warning: api_token in %s is a secret — prefer DEEPSOURCE_API_TOKEN env var\n", path)
	}
	if token := os.Getenv("DEEPSOURCE_API_TOKEN"); token != "" {
		cfg.DeepSource.APIToken = token
	}

	return &cfg, nil
}

// Validate validates the configuration.
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

// GenerateTemplate generates a configuration template.
func GenerateTemplate() string {
	return `# DSFix Configuration

deepsource:
  # Prefer setting DEEPSOURCE_API_TOKEN env var instead of putting the token here.
  api_token: ""

repository:
  # VCS owner/organization
  owner: ""
  # Repository name
  name: ""

filter:
  # Categories to include (leave empty for all)
  # Options: Bug Risk, Anti-pattern, Security, Performance, Typecheck, Style, Documentation
  categories: []

  # Severities to include (leave empty for all)
  # Options: critical, major, minor
  severities: []

  # Maximum number of issues to fetch (leave empty for unlimited)
  limit:

  # Optional path glob filters. Leave empty for all.
  # Examples:
  #   paths_include: ["internal/**", "cmd/**"]
  #   paths_exclude: ["vendor/**", "**/*_test.go"]
  paths_include: []
  paths_exclude: []

verify:
  # Command run after a fix to verify it compiles/passes. Leave empty to let the AI pick.
  # Examples: "go build ./...", "pnpm typecheck", "make lint"
  command: ""
`
}
