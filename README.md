# TailscaleKubeProxy

A secure Kubernetes API proxy using Tailscale's secure networking.

## Description

TailscaleKubeProxy is a lightweight bridge that connects your Tailscale network to your Kubernetes API. It allows you to securely manage your clusters from anywhere without exposing them to the internet.

For more information, see the blog post: [Kubernetes API access over Tailscale](https://0x2321.de/kubernetes-api-access-over-tailscale/)

### How it works
The proxy automatically maps Tailscale identities to Kubernetes users using impersonation. This means you can use your existing Kubernetes RBAC policies to control access based on Tailscale login names.

## Usage

### Running in Kubernetes

1. Create a Kubernetes service account with appropriate permissions for impersonation
2. Deploy the proxy as a pod in your cluster
3. Configure with your Tailscale auth key

Example Kubernetes deployment:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tailscale-kube-proxy
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tailscale-kube-proxy-impersonator
rules:
- apiGroups: [""]
  resources: ["users", "groups", "serviceaccounts"]
  verbs: ["impersonate"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tailscale-kube-proxy-impersonator
subjects:
- kind: ServiceAccount
  name: tailscale-kube-proxy
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: tailscale-kube-proxy-impersonator
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: Secret
metadata:
  name: tailscale-kube-proxy-secret
  namespace: kube-system
type: Opaque
stringData:
  authKey: "superSecret"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tailscale-kube-proxy
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tailscale-kube-proxy
  template:
    metadata:
      labels:
        app: tailscale-kube-proxy
    spec:
      serviceAccountName: tailscale-kube-proxy
      containers:
      - name: proxy
        image: ghcr.io/0x2321/tailscale-kube-proxy:latest
        env:
        - name: SECRET_NAME
          value: "tailscale-kube-proxy-secret"
        - name: HOSTNAME
          value: "awesome-cluster"
```

## Configuration

TailscaleKubeProxy can be configured using command-line flags or environment variables:

| Flag              | Environment Variable | Default                                                | Description                                                              |
|-------------------|----------------------|--------------------------------------------------------|--------------------------------------------------------------------------|
| `--hostname`      | `HOSTNAME`           | `kube-api`                                             | Hostname for this Tailscale node in the tailnet                          |
| `--secretName`    | `SECRET_NAME`        |                                                        | Name of the Kubernetes secret to store Tailscale state and auth key      |
| `--ephemeral`     | `EPHEMERAL`          | `true`                                                 | If true, the Tailscale node will be ephemeral                            |
| `--controlServer` | `CONTROL_SERVER`     | (defaults to Tailscale's servers if empty)             | URL of the Tailscale coordination server                                 |
| `--apiUrl`        | `API_URL`            | `https://kubernetes.default.svc`                       | URL of the Kubernetes API server to proxy requests to                    |
| `--tokenFile`     | `TOKEN_FILE`         | `/var/run/secrets/kubernetes.io/serviceaccount/token`  | Path to the Kubernetes service account token file                        |
| `--clusterCaFile` | `CLUSTER_CA_FILE`    | `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt` | Path to a file containing the Kubernetes API CA certificate              |
| `--insecure`      | `INSECURE`           | `false`                                                | If true, the Kubernetes API certificate will not be checked for validity |


### Building from Source

```bash
# Clone the repository
git clone https://github.com/0x2321/tailscale-kube-proxy.git
cd tailscale-kube-proxy

# Build the binary
go build -o tskp

# Or build the Docker image
docker build -t tailscale-kube-proxy .
```
