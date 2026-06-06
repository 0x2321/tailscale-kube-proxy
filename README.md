# TailscaleKubeProxy

A secure Kubernetes API proxy that bridges your [Tailscale](https://tailscale.com) network to your Kubernetes API server. It enables secure, zero-trust access to your clusters without exposing them to the public internet.

## 🚀 Features

- **Identity-based Access**: Maps Tailscale identities directly to Kubernetes users.
- **RBAC Integration**: Uses standard Kubernetes impersonation headers to leverage existing RBAC policies.
- **Zero Trust**: No public endpoints required; access is strictly via your encrypted Tailnet.
- **Easy Configuration**: Works with standard Kubernetes secrets and supports custom coordination servers (like Headscale).

## ⚠️ Important Note on Versioning

**The `latest` Docker tag is now deprecated.**

## 🛠 How it Works

TailscaleKubeProxy runs as a pod in your cluster and joins your Tailnet. When you access the proxy via its Tailscale IP or hostname:
1. The proxy identifies the source user using Tailscale's `WhoIs` API.
2. It forwards the request to the Kubernetes API server.
3. It adds impersonation headers (`Impersonate-User`, `Impersonate-Group`) based on the Tailscale identity.
4. Your existing Kubernetes RBAC determines if the user has permission for the requested action.

## 📦 Deployment

### 1. Prerequisites
- A Tailscale Auth Key (recommended: [tagged and non-expiring](https://tailscale.com/kb/1085/auth-keys/)).
- A Kubernetes cluster with RBAC enabled.

### 2. Kubernetes Installation

The proxy requires permissions to impersonate users and manage its own state in a Secret.

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
  name: tailscale-kube-proxy
  namespace: kube-system
type: Opaque
stringData:
  "TS_AUTHKEY": "tskey-auth-..." # Replace with your Tailscale Auth Key
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
        image: ghcr.io/0x2321/tailscale-kube-proxy:2
        args:
        - "--secret"
        - "tailscale-kube-proxy"
        - "--hostname"
        - "awesome-cluster"
```

## ⚙️ Configuration

TailscaleKubeProxy can be configured via command-line flags or environment variables. Environment variables are prefixed with nothing and use underscores (e.g., `--authkey` becomes `AUTHKEY`).

| Flag           | Env Var      | Default | Description                                   |
|----------------|--------------|---------|-----------------------------------------------|
| `--hostname`   | `HOSTNAME`   |         | Hostname for this node in your Tailnet        |
| `--secret`     | `SECRET`     |         | K8s secret name to store Tailscale state      |
| `--authkey`    | `AUTHKEY`    |         | Tailscale authentication key                  |
| `--controlurl` | `CONTROLURL` |         | Custom control URL (e.g., for Headscale)      |
| `--ephemeral`  | `EPHEMERAL`  | `false` | If true, node is removed when it goes offline |
| `--insecure`   | `INSECURE`   | `false` | Skip K8s API certificate verification         |

## 🔗 Resources

- [Blog Post: Kubernetes API access over Tailscale](https://0x2321.de/kubernetes-api-access-over-tailscale/)
- [Tailscale Documentation](https://tailscale.com/kb/)
