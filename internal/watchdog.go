package internal

import (
	"context"
	"fmt"
	"time"

	"tailscale.com/client/local"
)

const (
	// watchdogInterval is the time between health checks of the Tailscale backend.
	watchdogInterval = 30 * time.Second
	// statusTimeout is the maximum time allowed for a Tailscale status request.
	statusTimeout = 10 * time.Second
)

// startTailscaleWatchdog starts a background goroutine that periodically checks the status of the Tailscale backend.
// It returns a channel that will receive an error if the Tailscale backend is not running or if a status check fails.
// The watchdog stops when the provided context is canceled.
func startTailscaleWatchdog(ctx context.Context, lc *local.Client) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(watchdogInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := checkTailscaleStatus(ctx, lc); err != nil {
					errChan <- fmt.Errorf("tailscale watchdog check failed: %w", err)
					return
				}
			}
		}
	}()

	return errChan
}

// checkTailscaleStatus performs a single health check of the Tailscale backend.
// It returns an error if the status cannot be retrieved or if the backend state is not "Running".
func checkTailscaleStatus(ctx context.Context, lc *local.Client) error {
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()

	status, err := lc.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get backend status: %w", err)
	}
	if status.BackendState != "Running" {
		return fmt.Errorf("backend is in %q state, expected \"Running\"", status.BackendState)
	}

	return nil
}
