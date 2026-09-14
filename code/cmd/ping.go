package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/amx"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test network connection to the AMX matrix switcher",
	RunE: func(cmd *cobra.Command, args []string) error {
		amxClient, err := amx.NewClient(amx.Config{
			Host:     cfg.AMX.Host,
			Timeout:  cfg.AMX.Timeout,
			Insecure: cfg.AMX.Insecure,
			Headers:  cfg.AMX.Headers,
		})
		if err != nil {
			return fmt.Errorf("failed to create AMX client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.AMX.Timeout+2*time.Second)
		defer cancel()

		start := time.Now()
		err = amxClient.Ping(ctx)
		latency := time.Since(start).Round(time.Millisecond)

		if err != nil {
			return fmt.Errorf("ping failed to %s: %w", cfg.AMX.Host, err)
		}

		fmt.Printf("AMX DVX Switcher at %s is reachable! Latency: %s\n", cfg.AMX.Host, latency)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pingCmd)
}
