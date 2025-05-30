// TailscaleKubeProxy - A secure Kubernetes API proxy using Tailscale
//
// This application creates a secure gateway to a Kubernetes API server using Tailscale's
// secure networking. It authenticates users based on their Tailscale identity and maps
// those identities to Kubernetes users through impersonation.
//
// The proxy runs inside a Kubernetes cluster with appropriate service account permissions
// and exposes the Kubernetes API over Tailscale, allowing authorized Tailscale users
// to securely access the Kubernetes API without exposing it to the public internet.
package main

import "tailscaleKubeProxy/cmd"

func main() {
	cmd.Execute()
}
