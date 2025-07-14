// Package internal provides core functionality for the Tailscale Kubernetes Proxy.
package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"log"
	"os"
	"sync"
	"tailscale.com/ipn"
	"time"
)

// SecretStore provides persistent storage for Tailscale state using Kubernetes Secrets.
// It implements the necessary methods for Tailscale's state management while
// ensuring thread-safety and proper synchronization with the Kubernetes API.
type SecretStore struct {
	Data    map[string][]byte
	AuthKey string

	mutex     sync.Mutex
	client    *kubernetes.Clientset
	name      string
	namespace string
}

// NewSecretStore creates and initializes a new SecretStore with the given secret name.
// It sets up Kubernetes client connections and informers to watch for changes to the
// specified secret, ensuring the in-memory state stays synchronized with Kubernetes.
func NewSecretStore(name string) (*SecretStore, error) {
	store := &SecretStore{
		name: name,
		Data: make(map[string][]byte),
	}

	// Initialize Kubernetes in-cluster client configuration
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}
	store.client = client

	// Determine the current namespace from the service account
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Errorf("reading service account namespace: %v", err)
	}
	store.namespace = string(namespace)

	// Set up informers to watch for changes to the secret
	factory := informers.NewSharedInformerFactoryWithOptions(client, 30*time.Minute, informers.WithNamespace(string(namespace)))
	informer := factory.Core().V1().Secrets().Informer()

	// Handler function to process secret updates
	informerHandler := func(secret *corev1.Secret) {
		store.mutex.Lock()
		defer store.mutex.Unlock()

		store.AuthKey = string(secret.Data["authKey"])
		if err = json.Unmarshal(secret.Data["state"], &store.Data); err != nil {
			log.Printf("failed to unmarshal secret data: %v", err)
		}
	}

	// Register event handlers for secret creation and updates
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*corev1.Secret)
			if secret.Name == name {
				informerHandler(secret)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			secret := new.(*corev1.Secret)
			if secret.Name == name {
				informerHandler(secret)
			}
		},
	})

	// Start the informer and wait for initial sync
	factory.Start(wait.NeverStop)
	factory.WaitForCacheSync(wait.NeverStop)

	return store, nil
}

// updateSecret persists the current in-memory state to the Kubernetes Secret.
// It marshals the Data map to JSON, encodes it as base64, and applies a strategic
// merge patch to update only the data field of the secret.
func (s *SecretStore) updateSecret() error {
	newData, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}

	patch := []byte(`{"data":{"state":"` + base64.StdEncoding.EncodeToString(newData) + `"}}`)
	_, err = s.client.CoreV1().
		Secrets(s.namespace).
		Patch(context.Background(), s.name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})

	return err
}

// ReadState retrieves state data for the given Tailscale state key.
// It implements the ipn.StateStore interface required by Tailscale.
func (s *SecretStore) ReadState(id ipn.StateKey) ([]byte, error) {
	if data, ok := s.Data[string(id)]; ok {
		return data, nil
	}

	return nil, ipn.ErrStateNotExist
}

// WriteState updates the state data for the given Tailscale state key.
// It implements the ipn.StateStore interface required by Tailscale.
// The method is thread-safe and handles rollback in case of errors.
func (s *SecretStore) WriteState(id ipn.StateKey, bs []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	oldValue := s.Data[string(id)]
	s.Data[string(id)] = bs

	if err := s.updateSecret(); err != nil {
		s.Data[string(id)] = oldValue
		return err
	}

	return nil
}
