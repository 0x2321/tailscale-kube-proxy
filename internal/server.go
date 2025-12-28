package internal

// Package internal contains the core functionality of the TailscaleKubeProxy application.
// This package implements the secure proxy between Tailscale network and Kubernetes API server,
// handling authentication, authorization, and request forwarding.

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"tailscale.com/tsnet"
)

// RunServer is the entry point for the TailscaleKubeProxy application.
// It sets up a Tailscale node that acts as a secure proxy to a Kubernetes API server.
func RunServer(cmd *cobra.Command, args []string) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting TailscaleKubeProxy server...")

	// Log configuration parameters (without sensitive data)
	log.Printf("Configuration: API_URL=%s, TOKEN_FILE=%s, HOSTNAME=%s, EPHEMERAL=%v",
		viper.GetString("API_URL"),
		viper.GetString("TOKEN_FILE"),
		viper.GetString("HOSTNAME"),
		viper.GetBool("EPHEMERAL"))

	// Read the Kubernetes service account token from a file
	// This token is used to authenticate to the Kubernetes API server
	tokenFile := viper.GetString("TOKEN_FILE")
	serviceAccountToken, err := os.ReadFile(tokenFile)
	if err != nil {
		return fmt.Errorf("failed to read service account token from %q: %w", tokenFile, err)
	}

	// Initialize a new Tailscale server instance with configuration from environment
	// This creates an embedded Tailscale node that will join your tailnet
	s := new(tsnet.Server)
	s.Hostname = viper.GetString("HOSTNAME")
	s.Ephemeral = viper.GetBool("EPHEMERAL")
	s.ControlURL = viper.GetString("CONTROL_SERVER")

	if authKey := viper.GetString("AUTH_KEY"); authKey != "" {
		log.Println("WARNING: Using AUTH_KEY is deprecated, please use SECRET_NAME instead.")
		s.AuthKey = authKey
	}

	// Configure a secret store for Tailscale if SECRET_NAME is provided
	// This allows Tailscale to store its state in a Kubernetes secret
	if secretName := viper.GetString("SECRET_NAME"); secretName != "" {
		store, err := NewSecretStore(secretName)
		if err != nil {
			return fmt.Errorf("failed to initialize secret store for %q: %w", secretName, err)
		}
		s.AuthKey = store.AuthKey
		s.Store = store
	}

	defer s.Close()

	// Create a TCP listener on port 80 (standard HTTP port)
	// This listener will accept connections from other Tailscale nodes
	listenAddr := ":80"
	ln, err := s.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %q: %w", listenAddr, err)
	}
	defer ln.Close()

	// Get a local client to interact with the Tailscale node
	// This client allows us to perform operations like identifying connecting users
	lc, err := s.LocalClient()
	if err != nil {
		return fmt.Errorf("failed to get local Tailscale client: %w", err)
	}

	// Configure the target Kubernetes API server URL
	// For in-cluster operation, this is typically "https://kubernetes.default.svc"
	apiURL := viper.GetString("API_URL")
	kubernetesURL, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes API URL %q: %w", apiURL, err)
	}

	// Set up a reverse proxy to the Kubernetes API server
	proxy, err := newKubernetesProxy(kubernetesURL, lc, string(serviceAccountToken))
	if err != nil {
		return fmt.Errorf("failed to initialize kubernetes proxy: %w", err)
	}

	// Start the Tailscale connection
	if _, err := s.Up(cmd.Context()); err != nil {
		return fmt.Errorf("failed to connect to tailnet: %w", err)
	}

	// Start a watchdog to monitor Tailscale status
	tsError := startTailscaleWatchdog(cmd.Context(), lc)

	// Start the HTTP server
	ipv4, _ := s.TailscaleIPs()
	log.Printf("TailscaleKubeProxy is ready to serve requests at http://%s", ipv4.String())

	// Create a channel to listen for errors from the server
	serverError := make(chan error, 1)

	// Start the HTTP server in a goroutine
	go func() {
		serverError <- http.Serve(ln, proxy)
	}()

	// Wait for either the context to be canceled or the server to return an error
	select {
	case <-cmd.Context().Done():
		log.Println("Shutting down TailscaleKubeProxy server...")
		// The listener will be closed by the deferred ln.Close()
		// and the Tailscale server by the deferred s.Close()
		return nil
	case err := <-tsError:
		return fmt.Errorf("connection error: %v", err)
	case err := <-serverError:
		return fmt.Errorf("HTTP server error: %v", err)
	}
}
