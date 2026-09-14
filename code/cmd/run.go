package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/amx"
	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/bot"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Discord bot daemon",
	Long:  `Connects to Discord and registers slash commands to control the AMX video matrixer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Discord.Token == "" {
			return fmt.Errorf("discord token is required (set via --discord-token, DISCORD_TOKEN env var, or in config.yaml)")
		}

		log.Printf("Initializing AMX client for host: %s", cfg.AMX.Host)
		amxClient, err := amx.NewClient(amx.Config{
			Host:     cfg.AMX.Host,
			Timeout:  cfg.AMX.Timeout,
			Insecure: cfg.AMX.Insecure,
		})
		if err != nil {
			return fmt.Errorf("failed to create AMX client: %w", err)
		}

		b, err := bot.New(cfg, amxClient)
		if err != nil {
			return fmt.Errorf("failed to initialize Discord bot: %w", err)
		}

		if err := b.Start(); err != nil {
			return fmt.Errorf("failed to start Discord bot: %w", err)
		}

		log.Println("Discord ScreenBot is running! Press Ctrl+C to exit.")

		// Wait for OS termination signal
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-stopChan

		return b.Stop()
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
}
