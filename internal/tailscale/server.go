package tailscale

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/spf13/viper"
	"tailscale.com/client/local"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

// Server represents a Tailscale tsnet server instance.
type Server struct {
	ts     *tsnet.Server
	client *local.Client
	ln     net.Listener
}

// NewServer initializes and starts a new tsnet server using the provided Kubernetes store.
func NewServer(store *KubernetesStore) (*Server, error) {
	server := new(Server)

	// Check if authkey is set
	if viper.GetString("ts.authkey") == "" {
		return nil, fmt.Errorf("authkey is required")
	}

	// Create a new tsnet server
	server.ts = &tsnet.Server{
		Hostname:   viper.GetString("ts.hostname"),
		AuthKey:    viper.GetString("ts.authkey"),
		ControlURL: viper.GetString("ts.control_url"),
		Ephemeral:  viper.GetBool("ts.ephemeral"),
		Store:      store,
	}

	// Enable logging if debug flag is set
	if viper.GetBool("debug") {
		server.ts.Logf = logger.WithPrefix(log.Printf, "tsnet")
	}

	// Start the tsnet server
	if err := server.ts.Start(); err != nil {
		return nil, fmt.Errorf("failed to connect tsnet server: %w", err)
	}

	// Create a local client
	var err error
	server.client, err = server.ts.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create local client: %w", err)
	}

	// We listen on port 80 to provide a standard entry point for internal proxying
	// within the Tailscale network, regardless of the actual target service port.
	server.ln, err = server.ts.Listen("tcp", ":80")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port 80: %w", err)
	}

	return server, nil
}

// Listener returns the network listener for the tsnet server.
func (s *Server) Listener() net.Listener {
	return s.ln
}

// Close shuts down the tsnet server.
func (s *Server) Close() error {
	return s.ts.Close()
}

// UserProfile is a wrapper around tailcfg.UserProfile.
type UserProfile tailcfg.UserProfile

// WhoIs returns the profile of the user associated with the remote address.
func (s *Server) WhoIs(c context.Context, remoteAddr string) (*UserProfile, error) {
	resp, err := s.client.WhoIs(c, remoteAddr)
	if err != nil {
		return nil, err
	}

	return (*UserProfile)(resp.UserProfile), nil
}

// IsConnected returns true if the Tailscale client is connected to the Tailscale network.
func (s *Server) IsConnected(ctx context.Context) bool {
	status, err := s.client.StatusWithoutPeers(ctx)

	if err != nil {
		return false
	}

	return status.BackendState == "Running"
}
