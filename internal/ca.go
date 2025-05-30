package internal

// This file provides certificate authority (CA) functionality for the TailscaleKubeProxy.
// It handles the creation and management of certificate pools used for secure TLS connections
// to the Kubernetes API server.

import (
	"crypto/x509"
	"fmt"
	"github.com/spf13/viper"
	"os"
)

// getCaPool creates and returns a certificate pool for TLS connections.
// It starts with the system certificate pool (if available) and adds any
// additional CA certificates specified in the configuration.
//
// Returns:
//   - *x509.CertPool: The certificate pool containing system and custom CAs
//   - error: An error if loading or parsing certificates fails
func getCaPool() (*x509.CertPool, error) {
	caPool, _ := x509.SystemCertPool()
	if caPool == nil {
		caPool = x509.NewCertPool()
	}

	if filePath := viper.GetString("CLUSTER_CA_FILE"); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading CA file: %v", err)
		}

		if ok := caPool.AppendCertsFromPEM(data); !ok {
			return nil, fmt.Errorf("no certificates found in CA file")
		}
	}

	return caPool, nil
}
