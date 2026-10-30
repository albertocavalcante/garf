package config

import (
	"fmt"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/spf13/viper"
)

const defaultConcurrent = 4

// RegistryCredentials holds registry authentication information.
type RegistryCredentials struct {
	URL      string
	User     string
	Password string
	Type     string
}

// Builder provides a fluent interface for building configuration.
type Builder struct {
	config   *Config
	err      error
	vipers   map[string]*viper.Viper
	registry *RegistryCredentials
}

// NewBuilder creates a new configuration builder.
func NewBuilder() *Builder {
	return &Builder{
		config: &Config{
			LogLevel:   "info",
			Concurrent: defaultConcurrent,
		},
		vipers: make(map[string]*viper.Viper),
	}
}

// WithViper sets a viper instance for the given prefix.
func (b *Builder) WithViper(prefix string, v *viper.Viper) *Builder {
	if b.err != nil {
		return b
	}

	b.vipers[prefix] = v

	return b
}

// WithSource sets the source configuration.
func (b *Builder) WithSource(sourceURL, sourcePathStrip string) *Builder {
	if b.err != nil {
		return b
	}

	b.config.Source = SourceConfig{
		Type: core.DetectSourceType(sourceURL, sourcePathStrip),
		URL:  sourceURL,
	}

	return b
}

// WithDestination sets the destination configuration.
func (b *Builder) WithDestination(destPath, sourcePathStrip string) *Builder {
	if b.err != nil {
		return b
	}

	if b.registry == nil {
		b.err = fmt.Errorf("registry credentials must be set before destination")

		return b
	}

	b.config.Destination = DestinationConfig{
		Type:            b.registry.Type,
		URL:             b.registry.URL,
		User:            b.registry.User,
		Password:        b.registry.Password,
		DestPath:        destPath,
		SourcePathStrip: sourcePathStrip,
	}

	return b
}

// WithRegistryCredentials sets registry credentials.
func (b *Builder) WithRegistryCredentials(creds *RegistryCredentials) *Builder {
	if b.err != nil {
		return b
	}

	b.registry = creds

	return b
}

// Build validates and returns the configuration.
func (b *Builder) Build() (*Config, error) {
	if b.err != nil {
		return nil, b.err
	}

	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return b.config, nil
}

// RegistryConfig holds registry configuration parameters.
type RegistryConfig struct {
	URL               string
	User              string
	Password          string
	Type              string
	PasswordFromStdin bool
}

// JFrogConfig holds JFrog configuration parameters.
type JFrogConfig struct {
	URL               string
	User              string
	Password          string
	PasswordFromStdin bool
}

// CredentialsResolver handles credential resolution from various sources.
type CredentialsResolver struct {
	jfrogViper    *viper.Viper
	registryViper *viper.Viper
}

// NewCredentialsResolver creates a new credentials resolver.
func NewCredentialsResolver() *CredentialsResolver {
	jfrogViper := viper.New()
	jfrogViper.SetEnvPrefix("JFROG")
	jfrogViper.AutomaticEnv()

	registryViper := viper.New()
	registryViper.SetEnvPrefix("REGISTRY")
	registryViper.AutomaticEnv()

	return &CredentialsResolver{
		jfrogViper:    jfrogViper,
		registryViper: registryViper,
	}
}

// ResolveCredentials resolves registry credentials from flags and environment.
func (r *CredentialsResolver) ResolveCredentials(registryConfig *RegistryConfig, jfrogConfig *JFrogConfig) (*RegistryCredentials, error) {
	// Check if registry config is being used
	if r.isRegistryConfigProvided(registryConfig) {
		return r.resolveRegistryCredentials(registryConfig)
	}

	// Fall back to JFrog credentials
	return r.resolveJFrogCredentials(jfrogConfig)
}

func (r *CredentialsResolver) isRegistryConfigProvided(config *RegistryConfig) bool {
	flagsSet := config.URL != "" || config.User != "" || config.Password != "" ||
		config.PasswordFromStdin || config.Type != ""

	envSet := r.registryViper.GetString("URL") != "" ||
		r.registryViper.GetString("USER") != "" ||
		r.registryViper.GetString("PASSWORD") != ""

	return flagsSet || envSet
}

func (r *CredentialsResolver) resolveRegistryCredentials(config *RegistryConfig) (*RegistryCredentials, error) {
	creds := &RegistryCredentials{
		URL:  r.getStringValue(config.URL, r.registryViper.GetString("URL")),
		User: r.getStringValue(config.User, r.registryViper.GetString("USER")),
		Type: r.getStringValue(config.Type, "jfrog"), // Default to jfrog for backward compatibility
	}

	if config.PasswordFromStdin {
		// This would need to be handled by the caller since it requires IO
		creds.Password = config.Password // Will be set by caller
	} else {
		creds.Password = r.getStringValue(config.Password, r.registryViper.GetString("PASSWORD"))
	}

	// Normalize URL
	creds.URL = r.normalizeURL(creds.URL)

	return creds, r.validateRegistryCredentials(creds)
}

func (r *CredentialsResolver) resolveJFrogCredentials(config *JFrogConfig) (*RegistryCredentials, error) {
	creds := &RegistryCredentials{
		URL:  r.getStringValue(config.URL, r.jfrogViper.GetString("URL")),
		User: r.getStringValue(config.User, r.jfrogViper.GetString("USER")),
		Type: "jfrog",
	}

	if config.PasswordFromStdin {
		// This would need to be handled by the caller since it requires IO
		creds.Password = config.Password // Will be set by caller
	} else {
		creds.Password = r.getStringValue(config.Password, r.jfrogViper.GetString("PASSWORD"))
	}

	// Normalize URL
	creds.URL = r.normalizeURL(creds.URL)

	return creds, r.validateJFrogCredentials(creds)
}

func (r *CredentialsResolver) getStringValue(flag, env string) string {
	if flag != "" {
		return flag
	}

	return env
}

func (r *CredentialsResolver) normalizeURL(url string) string {
	if url == "" {
		return url
	}

	// Add scheme if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "http://" + url
	}

	return url
}

func (r *CredentialsResolver) validateRegistryCredentials(creds *RegistryCredentials) error {
	missing := []string{}

	if creds.URL == "" {
		missing = append(missing, "--registry-url or REGISTRY_URL")
	}

	if creds.User == "" {
		missing = append(missing, "--registry-user or REGISTRY_USER")
	}

	if creds.Password == "" {
		missing = append(missing, "--registry-password, --registry-password-stdin, or REGISTRY_PASSWORD")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing registry credentials: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (r *CredentialsResolver) validateJFrogCredentials(creds *RegistryCredentials) error {
	if creds.URL == "" {
		return fmt.Errorf("JFrog URL is required")
	}

	if creds.User == "" {
		return fmt.Errorf("JFrog user is required")
	}

	if creds.Password == "" {
		return fmt.Errorf("JFrog password is required")
	}

	return nil
}
