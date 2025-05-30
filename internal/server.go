package internal

// Package internal contains the core functionality of the TailscaleKubeProxy application.
// This package implements the secure proxy between Tailscale network and Kubernetes API server,
// handling authentication, authorization, and request forwarding.

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
	serviceAccountToken, err := os.ReadFile(viper.GetString("TOKEN_FILE"))
	if err != nil {
		return fmt.Errorf("failed to read service account token: %v", err)
	}

	// Initialize a new Tailscale server instance with configuration from environment
	// This creates an embedded Tailscale node that will join your tailnet
	s := new(tsnet.Server)
	s.Hostname = viper.GetString("HOSTNAME")
	s.Ephemeral = viper.GetBool("EPHEMERAL")
	s.ControlURL = viper.GetString("CONTROL_SERVER")
	s.AuthKey = viper.GetString("AUTH_KEY")
	defer s.Close()

	// Create a TCP listener on port 80 (standard HTTP port)
	// This listener will accept connections from other Tailscale nodes
	ln, err := s.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}
	defer ln.Close()

	// Get a local client to interact with the Tailscale node
	// This client allows us to perform operations like identifying connecting users
	lc, err := s.LocalClient()
	if err != nil {
		return fmt.Errorf("failed to get local Tailscale client: %v", err)
	}

	// Configure the target Kubernetes API server URL
	// For in-cluster operation, this is typically "https://kubernetes.default.svc"
	kubernetesURL, err := url.Parse(viper.GetString("API_URL"))
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes URL: %v", err)
	}

	// Set up a reverse proxy to the Kubernetes API server
	proxy := httputil.NewSingleHostReverseProxy(kubernetesURL)
	originalDirector := proxy.Director

	// Retrieve the certificate authority pool for secure TLS connections
	// This includes system certificates and any custom CA certificates specified in configuration
	caPool, err := getCaPool()
	if err != nil {
		return fmt.Errorf("failed to import certificates: %v", err)
	}

	// Configure the HTTP transport with TLS settings for secure communication with the Kubernetes API server
	// This sets up the root certificate authorities and handles the insecure flag option
	// which can be used to skip certificate validation in development environments
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caPool,
			InsecureSkipVerify: viper.GetBool("INSECURE"),
		},
	}

	// Configure the proxy director to handle authentication and user impersonation
	// This maps Tailscale identities to Kubernetes RBAC permissions
	proxy.Director = func(r *http.Request) {
		originalDirector(r)

		// Clear any existing impersonation headers
		r.Header.Del("Impersonate-User")
		r.Header.Del("Impersonate-Group")

		// Identify the Tailscale user making the request based on their IP
		who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
		if err == nil {
			log.Printf("%s %s user=%s ip=%s", r.Method, r.URL.Path, who.UserProfile.LoginName, r.RemoteAddr)

			// Set Kubernetes impersonation headers to enable RBAC based on Tailscale identity
			// See: https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation
			r.Header.Set("Impersonate-User", who.UserProfile.LoginName)
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", serviceAccountToken))
		} else {
			log.Printf("%s %s user=unknown ip=%s", r.Method, r.URL.Path, r.RemoteAddr)
		}
	}

	// Start the Tailscale connection
	_, err = s.Up(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to connect to tailnet: %v", err)
	}

	// Start the HTTP server
	ipv4, _ := s.TailscaleIPs()
	log.Printf("TailscaleKubeProxy is ready to serve requests at http://%s", ipv4.String())
	return http.Serve(ln, proxy)
}
