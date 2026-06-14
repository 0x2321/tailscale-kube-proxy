package tailscale

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"tailscale.com/ipn"
)

// KubernetesStore implements ipn.StateStore by persisting state in a Kubernetes Secret.
// It maintains an in-memory cache to avoid frequent API calls for reads.
type KubernetesStore struct {
	state     map[ipn.StateKey][]byte
	client    *kubernetes.Clientset
	namespace string
	secret    string
	mu        sync.RWMutex
}

// NewKubernetesStore initializes a new store and loads existing state from the specified Secret.
func NewKubernetesStore(namespace string, secret string, config *rest.Config) (ipn.StateStore, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	store := &KubernetesStore{
		state:     make(map[ipn.StateKey][]byte),
		client:    clientset,
		namespace: namespace,
		secret:    secret,
	}
	if err = store.initStore(); err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	return store, nil
}

// initStore populates the in-memory cache from the Kubernetes Secret.
func (s *KubernetesStore) initStore() error {
	secret, err := s.client.
		CoreV1().
		Secrets(s.namespace).
		Get(context.TODO(), s.secret, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	for k, v := range secret.Data {
		s.state[ipn.StateKey(k)] = v
	}

	return nil
}

// ReadState returns the state for the given key from the local cache.
func (s *KubernetesStore) ReadState(id ipn.StateKey) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if bs, ok := s.state[id]; ok {
		return bs, nil
	}
	return nil, ipn.ErrStateNotExist
}

// WriteState updates the local cache and persists the change to Kubernetes.
func (s *KubernetesStore) WriteState(id ipn.StateKey, bs []byte) error {
	s.mu.Lock()
	s.state[id] = bs
	s.mu.Unlock()

	// Use a Strategic Merge Patch to update only the specific key in the Secret's data.
	// This avoids race conditions and unnecessary overhead of fetching the full Secret first.
	patchData := map[string]interface{}{
		"data": map[string]string{
			// Values in a Secret's 'data' field must be base64 encoded when using Patch.
			string(id): base64.StdEncoding.EncodeToString(bs),
		},
	}
	payloadBytes, _ := json.Marshal(patchData)

	_, err := s.client.CoreV1().Secrets(s.namespace).Patch(
		context.TODO(),
		s.secret,
		types.StrategicMergePatchType,
		payloadBytes,
		metav1.PatchOptions{},
	)
	return err
}
