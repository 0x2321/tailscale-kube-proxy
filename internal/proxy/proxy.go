package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"tailscale-kube-proxy/internal/tailscale"

	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
)

// ReverseProxy handles requests between Tailscale and the Kubernetes API.
type ReverseProxy struct {
	target *url.URL
	config *rest.Config
	http   *httputil.ReverseProxy
	ts     *tailscale.Server
}

// NewKubeProxy creates a new proxy instance with specialized TLS and rewrite logic.
func NewKubeProxy(config *rest.Config, ts *tailscale.Server) (*ReverseProxy, error) {
	proxy := &ReverseProxy{
		http:   &httputil.ReverseProxy{},
		ts:     ts,
		config: config,
	}

	targetUrl, err := url.Parse(config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}
	proxy.target = targetUrl
	proxy.http.Rewrite = proxy.rewrite

	// Manually build the CA pool to ensure we trust the Kubernetes API server
	// even if it uses a self-signed certificate provided via the rest.Config.
	caPool := x509.NewCertPool()
	if len(config.TLSClientConfig.CAData) > 0 {
		caPool.AppendCertsFromPEM(config.TLSClientConfig.CAData)
	} else if config.TLSClientConfig.CAFile != "" {
		ca, err := os.ReadFile(config.TLSClientConfig.CAFile)
		if err == nil {
			caPool.AppendCertsFromPEM(ca)
		}
	}

	// Configure the HTTP transport with TLS settings for secure communication with the Kubernetes API server
	// This sets up the root certificate authorities and handles the insecure flag option
	// which can be used to skip certificate validation in development environments
	proxy.http.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caPool,
			InsecureSkipVerify: viper.GetBool("insecure"),
		},
	}

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
	req.SetXForwarded()

	if user, err := r.ts.WhoIs(req.Out.Context(), req.In.RemoteAddr); err == nil {
		// Bridge Tailscale identity to Kubernetes by using the proxy's own token
		// and adding impersonation headers for the identified user.
		req.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.BearerToken))
		req.Out.Header.Set("Impersonate-Uid", user.ID.String())
		req.Out.Header.Set("Impersonate-User", user.LoginName)
		for _, group := range user.Groups {
			req.Out.Header.Add("Impersonate-Group", group)
		}

		log.Printf("%s %s user=%s ip=%s", req.In.Method, req.In.URL.Path, user.LoginName, req.In.RemoteAddr)
	} else {
		log.Printf("Warning: failed to identify Tailscale user for %s: %v", req.In.RemoteAddr, err)
		log.Printf("%s %s user=unknown ip=%s", req.In.Method, req.In.URL.Path, req.In.RemoteAddr)
	}
}

// Listen starts the proxy server on the Tailscale listener.
func (r *ReverseProxy) Listen() error {
	return http.Serve(r.ts.Listener(), r.http)
}
