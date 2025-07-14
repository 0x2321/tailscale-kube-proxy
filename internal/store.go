package internal

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"tailscale.com/ipn"
)

type SecretStore struct {
	client    *kubernetes.Clientset
	namespace string
	name      string
}

func NewSecretStore(name string) (*SecretStore, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}

	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Errorf("reading service account namespace: %v", err)
	}

	return &SecretStore{client, string(namespace), name}, nil
}

func (s *SecretStore) getSecret() (*corev1.Secret, error) {
	return s.client.CoreV1().
		Secrets(s.namespace).
		Get(context.TODO(), s.name, metav1.GetOptions{})
}

func (s *SecretStore) GetAuthKey() (string, error) {
	secret, err := s.getSecret()
	if err != nil {
		return "", err
	}

	return string(secret.Data["authKey"]), nil
}

func (s *SecretStore) ReadState(id ipn.StateKey) ([]byte, error) {
	secret, err := s.getSecret()
	if err != nil {
		return nil, err
	}

	return secret.Data[string(id)], nil
}

func (s *SecretStore) WriteState(id ipn.StateKey, bs []byte) error {
	secret, err := s.getSecret()
	if err != nil {
		return err
	}

	secret.Data[string(id)] = bs

	_, err = s.client.CoreV1().
		Secrets(s.namespace).
		Update(context.TODO(), secret, metav1.UpdateOptions{})
	return err
}
