/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"tailscale-kube-proxy/internal/proxy"
	"tailscale-kube-proxy/internal/tailscale"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"tailscale.com/ipn"
)

var (
	version = "development"
	time    = "n/a"
	commit  = "n/a"
)

func stringWithDefault(val, defaultValue string) string {
	if val == "" {
		return defaultValue
	}
	return val
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tailscale-kube-proxy",
	Short: "A proxy to connect Kubernetes services to a Tailscale network",
	Long: `tailscale-kube-proxy is a tool that allows you to expose Kubernetes services 
to your Tailscale network or vice-versa, providing a secure way to access 
in-cluster resources.`,
	Version: fmt.Sprintf("%s (commit: %s, built at %s)",
		stringWithDefault(version, "development"),
		stringWithDefault(commit, "n/a"),
		stringWithDefault(time, "n/a"),
	),
	RunE: run,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tailscale-kube-proxy.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().String("secret-name", "", "Name of the Kubernetes secret to store Tailscale state")
	_ = viper.BindPFlag("secret_name", rootCmd.Flags().Lookup("secret-name"))

	rootCmd.Flags().String("hostname", "kube-proxy", "Hostname to use for the Tailscale node")
	_ = viper.BindPFlag("ts.hostname", rootCmd.Flags().Lookup("hostname"))

	rootCmd.Flags().String("authkey", "", "Tailscale authentication key")
	_ = viper.BindPFlag("ts.authkey", rootCmd.Flags().Lookup("authkey"))

	rootCmd.Flags().String("control-url", "", "Custom Tailscale control URL (e.g. for Headscale)")
	_ = viper.BindPFlag("ts.control_url", rootCmd.Flags().Lookup("control-url"))

	rootCmd.Flags().Bool("ephemeral", false, "Whether to use an ephemeral Tailscale node")
	_ = viper.BindPFlag("ts.ephemeral", rootCmd.Flags().Lookup("ephemeral"))

	rootCmd.Flags().Bool("insecure", false, "Allow insecure connection to the Kubernetes API")
	_ = viper.BindPFlag("insecure", rootCmd.Flags().Lookup("insecure"))

	rootCmd.Flags().Bool("debug", false, "Enable debug logging")
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}

func run(cmd *cobra.Command, args []string) error {
	println(cmd.Version)
	if stringWithDefault(version, "development") == "development" {
		viper.Set("debug", true)
	}

	// kubernetes client config
	log.Println("Starting TailscaleKubeProxy server...")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	// initialize state store
	secretName := viper.GetString("secret_name")
	var store ipn.StateStore
	if secretName != "" {
		log.Printf("Using Kubernetes secret state store %s", secretName)
		nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			log.Fatalf("Failed to read namespace: %v", err)
		}

		store, err = tailscale.NewKubernetesStore(string(nsBytes), secretName, config)
		if err != nil {
			log.Fatalf("Failed to create store: %v", err)
		}
	}

	// initialize tailscale server
	ts, err := tailscale.NewServer(store)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer ts.Close()

	// initialize proxy
	server, err := proxy.NewKubeProxy(config, ts)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// start proxy
	return server.Listen()
}
