package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"tailscale-kube-proxy/internal/tailscale"

	"k8s.io/client-go/rest"
)

// ReverseProxy handles requests between Tailscale and the Kubernetes API.
type ReverseProxy struct {
	target *url.URL
	http   *httputil.ReverseProxy
	ts     *tailscale.Server
}

// NewKubeProxy creates a new proxy instance with specialized TLS and rewrite logic.
func NewKubeProxy(config *rest.Config, ts *tailscale.Server) (*ReverseProxy, error) {
	proxy := &ReverseProxy{
		http: &httputil.ReverseProxy{},
		ts:   ts,
	}

	// Parse the target URL.
	targetUrl, err := url.Parse(config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	// Rewrite the URL to the Kubernetes API server.
	proxy.target = targetUrl
	proxy.http.Rewrite = proxy.rewrite

	// Use the same configuration as the Kubernetes client.
	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, err
	}
	proxy.http.Transport = transport

	return proxy, nil
}

func (r *ReverseProxy) rewrite(req *httputil.ProxyRequest) {
	req.SetURL(r.target)
	req.Out.Host = r.target.Host
	req.Out.Header = make(http.Header)

	// Stripping incoming impersonation headers to prevent users from spoofing identities.
	// We only allow identities verified by the Tailscale 'WhoIs' check.
	for k, v := range req.In.Header {
		lowercaseKey := strings.ToLower(k)
		if strings.HasPrefix(lowercaseKey, "impersonate-") {
			continue
		}
		req.Out.Header[k] = v
	}

	if user, err := r.ts.WhoIs(req.Out.Context(), req.In.RemoteAddr); err == nil {
		// Bridge Tailscale identity to Kubernetes by using the proxy's own token
		// and adding impersonation headers for the identified user.
		req.Out.Header.Set("Impersonate-Uid", user.ID.String())
		req.Out.Header.Set("Impersonate-User", user.LoginName)
		for _, group := range user.Groups {
			req.Out.Header.Add("Impersonate-Group", group)
		}

		log.Printf("%s %s user=%s ip=%s", req.In.Method, req.In.URL.Path, user.LoginName, req.In.RemoteAddr)
	} else {
		req.Out.Header.Set("Impersonate-User", "system:anonymous")
		log.Printf("Warning: failed to identify Tailscale user for %s: %v", req.In.RemoteAddr, err)
		log.Printf("%s %s user=unknown ip=%s", req.In.Method, req.In.URL.Path, req.In.RemoteAddr)
	}
}

// Listen starts the proxy server on the Tailscale listener.
func (r *ReverseProxy) Listen() error {
	log.Println("Starting proxy server...")
	return http.Serve(r.ts.Listener(), r.http)
}
