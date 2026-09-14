package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/amx"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <screen/output> <input>",
	Short: "Switch an output to an input on the AMX matrixer directly from CLI",
	Long: `Directly route a video input to an output without needing Discord.
Supports configured aliases (e.g. 'screena', 'laptop') as well as numeric port IDs.

Example:
  screenbot switch screena input2
  screenbot switch 1 4`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawScreen := args[0]
		rawInput := args[1]

		outputID, outName, err := cfg.ResolveOutput(rawScreen)
		if err != nil {
			return fmt.Errorf("invalid output: %w", err)
		}

		inputID, inName, err := cfg.ResolveInput(rawInput)
		if err != nil {
			return fmt.Errorf("invalid input: %w", err)
		}

		amxClient, err := amx.NewClient(amx.Config{
			Host:     cfg.AMX.Host,
			Timeout:  cfg.AMX.Timeout,
			Insecure: cfg.AMX.Insecure,
		})
		if err != nil {
			return fmt.Errorf("failed to create AMX client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.AMX.Timeout+2*time.Second)
		defer cancel()

		fmt.Printf("Routing %s (Port %d) -> %s (Port %d) on %s...\n",
			inName, inputID, outName, outputID, cfg.AMX.Host)

		if err := amxClient.SwitchVideo(ctx, outputID, inputID); err != nil {
			return fmt.Errorf("failed to switch: %w", err)
		}

		fmt.Printf("Successfully routed %s to %s!\n", inName, outName)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(switchCmd)
}
