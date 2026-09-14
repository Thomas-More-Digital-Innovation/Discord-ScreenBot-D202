package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Discord DiscordConfig `mapstructure:"discord"`
	AMX     AMXConfig     `mapstructure:"amx"`
	Mapping MappingConfig `mapstructure:"mapping"`
}

// DiscordConfig contains settings for Discord bot integration.
type DiscordConfig struct {
	Token     string `mapstructure:"token"`
	AppID     string `mapstructure:"app_id"`
	GuildID   string `mapstructure:"guild_id"`
	ChannelID string `mapstructure:"channel_id"`
}

// AMXConfig contains connection settings for the AMX matrix switcher.
type AMXConfig struct {
	Host     string        `mapstructure:"host"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Insecure bool          `mapstructure:"insecure"`
}

// MappingConfig maps user-friendly names to numeric matrix channels.
type MappingConfig struct {
	Outputs map[string]int `mapstructure:"outputs"`
	Inputs  map[string]int `mapstructure:"inputs"`
}

// Choice represents an option for slash command choices / autocomplete.
type Choice struct {
	Name  string
	Value string
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Discord: DiscordConfig{},
		AMX: AMXConfig{
			Host:     "http://192.168.1.73",
			Timeout:  5 * time.Second,
			Insecure: true,
		},
		Mapping: MappingConfig{
			Outputs: map[string]int{
				"screena": 1,
				"screenb": 2,
				"output1": 1,
				"output2": 2,
				"output3": 3,
				"output4": 4,
			},
			Inputs: map[string]int{
				"input1": 1,
				"input2": 2,
				"input3": 3,
				"input4": 4,
				"input5": 5,
				"input6": 6,
				"input7": 7,
				"input8": 8,
				"input9": 9,
				"input10": 10,
			},
		},
	}
}

// Load loads configuration from Viper using optional file path and env variables.
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Environment variable overrides
	v.SetEnvPrefix("SCREENBOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Direct environment variable fallbacks
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		v.Set("discord.token", token)
	}
	if guildID := os.Getenv("DISCORD_GUILD_ID"); guildID != "" {
		v.Set("discord.guild_id", guildID)
	}
	if channelID := os.Getenv("DISCORD_CHANNEL_ID"); channelID != "" {
		v.Set("discord.channel_id", channelID)
	}
	if amxHost := os.Getenv("AMX_HOST"); amxHost != "" {
		v.Set("amx.host", amxHost)
	}

	// Set defaults
	def := DefaultConfig()
	v.SetDefault("amx.host", def.AMX.Host)
	v.SetDefault("amx.timeout", "5s")
	v.SetDefault("amx.insecure", def.AMX.Insecure)
	v.SetDefault("mapping.outputs", def.Mapping.Outputs)
	v.SetDefault("mapping.inputs", def.Mapping.Inputs)

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", configFile, err)
		}
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		// Ignore error if file doesn't exist; rely on defaults / env
		_ = v.ReadInConfig()
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Ensure maps are initialized even if empty in file
	if cfg.Mapping.Outputs == nil {
		cfg.Mapping.Outputs = make(map[string]int)
	}
	if cfg.Mapping.Inputs == nil {
		cfg.Mapping.Inputs = make(map[string]int)
	}

	return cfg, nil
}

// NormalizeKey removes leading/trailing spaces and lowercases the input.
func NormalizeKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// ResolveOutput maps a user string (alias or number) to an output channel ID.
func (c *Config) ResolveOutput(raw string) (int, string, error) {
	norm := NormalizeKey(raw)
	if norm == "" {
		return 0, "", fmt.Errorf("output cannot be empty")
	}

	// Check mapping
	for alias, id := range c.Mapping.Outputs {
		if NormalizeKey(alias) == norm {
			return id, alias, nil
		}
	}

	// Check if integer
	if num, err := strconv.Atoi(norm); err == nil && num > 0 {
		return num, fmt.Sprintf("Output %d", num), nil
	}

	// Check if user passed "screen1", "output2", etc.
	for _, prefix := range []string{"screen", "output", "out"} {
		if strings.HasPrefix(norm, prefix) {
			rest := strings.TrimPrefix(norm, prefix)
			if num, err := strconv.Atoi(rest); err == nil && num > 0 {
				return num, fmt.Sprintf("Output %d", num), nil
			}
		}
	}

	return 0, "", fmt.Errorf("unknown output %q. Available: %s", raw, strings.Join(c.GetOutputNames(), ", "))
}

// ResolveInput maps a user string (alias or number) to an input channel ID.
func (c *Config) ResolveInput(raw string) (int, string, error) {
	norm := NormalizeKey(raw)
	if norm == "" {
		return 0, "", fmt.Errorf("input cannot be empty")
	}

	// Check mapping
	for alias, id := range c.Mapping.Inputs {
		if NormalizeKey(alias) == norm {
			return id, alias, nil
		}
	}

	// Check if integer
	if num, err := strconv.Atoi(norm); err == nil && num >= 0 {
		return num, fmt.Sprintf("Input %d", num), nil
	}

	// Check if user passed "in1", "input2", etc.
	for _, prefix := range []string{"input", "in"} {
		if strings.HasPrefix(norm, prefix) {
			rest := strings.TrimPrefix(norm, prefix)
			if num, err := strconv.Atoi(rest); err == nil && num >= 0 {
				return num, fmt.Sprintf("Input %d", num), nil
			}
		}
	}

	return 0, "", fmt.Errorf("unknown input %q. Available: %s", raw, strings.Join(c.GetInputNames(), ", "))
}

// GetOutputNames returns a sorted list of configured output aliases.
func (c *Config) GetOutputNames() []string {
	names := make([]string, 0, len(c.Mapping.Outputs))
	for k := range c.Mapping.Outputs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// GetInputNames returns a sorted list of configured input aliases.
func (c *Config) GetInputNames() []string {
	names := make([]string, 0, len(c.Mapping.Inputs))
	for k := range c.Mapping.Inputs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// GetOutputChoices returns options formatted for Discord slash command choices (max 25).
func (c *Config) GetOutputChoices() []Choice {
	names := c.GetOutputNames()
	if len(names) > 25 {
		names = names[:25]
	}
	choices := make([]Choice, len(names))
	for i, n := range names {
		choices[i] = Choice{
			Name:  fmt.Sprintf("%s (Port %d)", n, c.Mapping.Outputs[n]),
			Value: n,
		}
	}
	return choices
}

// GetInputChoices returns options formatted for Discord slash command choices (max 25).
func (c *Config) GetInputChoices() []Choice {
	names := c.GetInputNames()
	if len(names) > 25 {
		names = names[:25]
	}
	choices := make([]Choice, len(names))
	for i, n := range names {
		choices[i] = Choice{
			Name:  fmt.Sprintf("%s (Port %d)", n, c.Mapping.Inputs[n]),
			Value: n,
		}
	}
	return choices
}
