package cmd

import (
	"fmt"
	"os"

	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	amxHostFlag   string
	discordToken  string
	guildIDFlag   string
	channelIDFlag string
	cfg           *config.Config
)

// RootCmd is the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "screenbot",
	Short: "Discord bot and CLI tool to control AMX DVX-3150HD-SP video matrixers",
	Long: `ScreenBot allows routing video matrix outputs and inputs on an AMX DVX-3150HD-SP
either via Discord slash commands (/video screena input2) or directly from the CLI.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.yaml)")
	RootCmd.PersistentFlags().StringVar(&amxHostFlag, "amx-host", "", "AMX switcher host (e.g. http://192.168.1.73)")
	RootCmd.PersistentFlags().StringVar(&discordToken, "discord-token", "", "Discord bot token")
	RootCmd.PersistentFlags().StringVar(&guildIDFlag, "guild-id", "", "Discord guild ID for instant slash command registration")
	RootCmd.PersistentFlags().StringVar(&channelIDFlag, "channel-id", "", "Discord channel ID to restrict bot commands to")

	_ = viper.BindPFlag("amx.host", RootCmd.PersistentFlags().Lookup("amx-host"))
	_ = viper.BindPFlag("discord.token", RootCmd.PersistentFlags().Lookup("discord-token"))
	_ = viper.BindPFlag("discord.guild_id", RootCmd.PersistentFlags().Lookup("guild-id"))
	_ = viper.BindPFlag("discord.channel_id", RootCmd.PersistentFlags().Lookup("channel-id"))
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load configuration: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Apply CLI flag overrides if explicitly passed
	if amxHostFlag != "" {
		cfg.AMX.Host = amxHostFlag
	}
	if discordToken != "" {
		cfg.Discord.Token = discordToken
	}
	if guildIDFlag != "" {
		cfg.Discord.GuildID = guildIDFlag
	}
	if channelIDFlag != "" {
		cfg.Discord.ChannelID = channelIDFlag
	}
}
