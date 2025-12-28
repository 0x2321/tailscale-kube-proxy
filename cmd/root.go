package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tskp",
	Short: "A secure Kubernetes API proxy over Tailscale",
	Long: `TailscaleKubeProxy provides secure access to a Kubernetes API server 
over Tailscale network without exposing it to the public internet.

It creates a Tailscale node that acts as a reverse proxy to your Kubernetes API,
mapping Tailscale identities to Kubernetes identities for authentication and
authorization. This allows you to securely access your Kubernetes cluster
from anywhere using your Tailscale credentials.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
}
