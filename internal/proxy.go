package internal

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/spf13/viper"
	"tailscale.com/client/local"
)

// newKubernetesProxy creates a new reverse proxy that forwards requests to the Kubernetes API server.
// It handles TLS configuration, including custom CAs and insecure mode,
// and adds impersonation headers based on the Tailscale identity of the caller.
func newKubernetesProxy(target *url.URL, lc *local.Client, token string) (*httputil.ReverseProxy, error) {
	// Retrieve the certificate authority pool for secure TLS connections
	// This includes system certificates and any custom CA certificates specified in configuration
	caPool, err := getCaPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate authority pool: %w", err)
	}

	// Configure the proxy to handle authentication and user impersonation
	proxy := &httputil.ReverseProxy{
		// This maps Tailscale identities to Kubernetes RBAC permissions
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)

			// Clear any existing impersonation headers to prevent header injection
			// TODO: Is this needed with Rewrite?
			r.Out.Header.Del("Impersonate-User")
			r.Out.Header.Del("Impersonate-Group")

			// Identify the Tailscale user making the request based on their IP
			who, err := lc.WhoIs(r.Out.Context(), r.In.RemoteAddr)
			if err == nil {
				log.Printf("%s %s user=%s ip=%s", r.In.Method, r.In.URL.Path, who.UserProfile.LoginName, r.In.RemoteAddr)

				// Set Kubernetes impersonation headers to enable RBAC based on Tailscale identity
				// See: https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation
				r.Out.Header.Set("Impersonate-User", who.UserProfile.LoginName)
				r.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			} else {
				log.Printf("Warning: failed to identify Tailscale user for %s: %v", r.In.RemoteAddr, err)
				log.Printf("%s %s user=unknown ip=%s", r.In.Method, r.In.URL.Path, r.In.RemoteAddr)
			}

			r.SetXForwarded()
		},

		// Configure the HTTP transport with TLS settings for secure communication with the Kubernetes API server
		// This sets up the root certificate authorities and handles the insecure flag option
		// which can be used to skip certificate validation in development environments
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caPool,
				InsecureSkipVerify: viper.GetBool("INSECURE"),
			},
		},
	}

	return proxy, nil
}
