package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"tailscaleKubeProxy/internal"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Tailscale Kubernetes API proxy server",
	Long: `Start a Tailscale node that acts as a secure proxy to a Kubernetes API server.

This command initializes a Tailscale node, sets up a reverse proxy to the 
Kubernetes API server and handles authentication and identity mapping between
Tailscale users and Kubernetes RBAC.`,
	RunE: internal.RunServer,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	serveCmd.Flags().String("apiUrl", "https://kubernetes.default.svc", "URL of the Kubernetes API server to proxy requests to")
	_ = viper.BindPFlag("API_URL", serveCmd.Flags().Lookup("apiUrl"))

	serveCmd.Flags().String("tokenFile", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Path to the Kubernetes service account token file used for authentication to the Kubernetes API server")
	_ = viper.BindPFlag("TOKEN_FILE", serveCmd.Flags().Lookup("tokenFile"))

	serveCmd.Flags().Bool("insecure", false, "If true, the Kubernetes API certificate will not be checked for validity")
	_ = viper.BindPFlag("INSECURE", serveCmd.Flags().Lookup("insecure"))

	serveCmd.Flags().String("clusterCaFile", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "Path to a directory containing the Kubernetes API CA certificate")
	_ = viper.BindPFlag("CLUSTER_CA_FILE", serveCmd.Flags().Lookup("clusterCaFile"))

	serveCmd.Flags().String("hostname", "kube-api", "Hostname for this Tailscale node in the tailnet")
	_ = viper.BindPFlag("HOSTNAME", serveCmd.Flags().Lookup("hostname"))

	serveCmd.Flags().String("authKey", "", "(DEPRECATED USE secretName) Tailscale authentication key used to register this node with your tailnet")
	_ = viper.BindPFlag("AUTH_KEY", serveCmd.Flags().Lookup("authKey"))

	serveCmd.Flags().Bool("ephemeral", true, "If true, the Tailscale node will be ephemeral (not saved in the node list after shutdown)")
	_ = viper.BindPFlag("EPHEMERAL", serveCmd.Flags().Lookup("ephemeral"))

	serveCmd.Flags().String("controlServer", "", "URL of the Tailscale coordination server (defaults to Tailscale's servers if empty)")
	_ = viper.BindPFlag("CONTROL_SERVER", serveCmd.Flags().Lookup("controlServer"))

	serveCmd.Flags().String("secretName", "", "Name of the Kubernetes secret to store Tailscale state")
	_ = viper.BindPFlag("SECRET_NAME", serveCmd.Flags().Lookup("secretName"))

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
