package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var screensCmd = &cobra.Command{
	Use:   "screens",
	Short: "List all configured screen outputs and video inputs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AMX Switcher Host: %s\n\n", cfg.AMX.Host)

		fmt.Println("Configured Outputs (Screens):")
		for _, name := range cfg.GetOutputNames() {
			fmt.Printf("  • %-15s -> Output Port %d\n", name, cfg.Mapping.Outputs[name])
		}

		fmt.Println("\nConfigured Inputs (Sources):")
		for _, name := range cfg.GetInputNames() {
			fmt.Printf("  • %-15s -> Input Port %d\n", name, cfg.Mapping.Inputs[name])
		}
	},
}

func init() {
	RootCmd.AddCommand(screensCmd)
}
