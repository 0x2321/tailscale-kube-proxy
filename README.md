# TailscaleKubeProxy

A secure Kubernetes API proxy that bridges your [Tailscale](https://tailscale.com) network to your Kubernetes API server. It enables secure, zero-trust access to your clusters without exposing them to the public internet.

## 🚀 Features

- **Identity-based Access**: Maps Tailscale identities directly to Kubernetes users.
- **RBAC Integration**: Uses standard Kubernetes impersonation headers to leverage existing RBAC policies.
- **Zero Trust**: No public endpoints required; access is strictly via your encrypted Tailnet.
- **Easy Configuration**: Works with standard Kubernetes secrets and supports custom coordination servers (like Headscale).

## ⚠️ Important Note on Versioning

**The `latest` Docker tag is now deprecated.**
Starting with tag `2`, the image repository has changed to `codeberg.org/0x2321/tailscale-kube-proxy`.

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

### 2. Helm Installation

Add the Helm repository:

```bash
helm repo add 0x2321 https://codeberg.org/api/packages/0x2321/helm
helm repo update
```

Install the chart:

```bash
helm install tailscale-kube-proxy 0x2321/tailscale-kube-proxy \
  --set ts.authKey="YOUR_AUTH_KEY" \
  --set ts.hostname="awesome-cluster"
```

## ⚙️ Configuration

Configuration can be done via Helm values, environment variables, or CLI arguments.

| Helm Value      | Environment Variable | CLI Argument    | Default      | Description                                            |
|-----------------|----------------------|-----------------|--------------|--------------------------------------------------------|
| `ts.hostname`   | `TS_HOSTNAME`        | `--hostname`    | `kubernetes` | Hostname for this node in the Tailnet                  |
| `ts.authKey`    | `TS_AUTHKEY`         | `--authkey`     |              | Tailscale Authentication Key                           |
| `ts.controlUrl` | `TS_CONTROL_URL`     | `--control-url` |              | Custom control URL (e.g., for Headscale)               |
| `ts.ephemeral`  | `TS_EPHEMERAL`       | `--ephemeral`   | `false`      | If true, the node is removed when going offline        |
| -               | `SECRET_NAME`        | `--secret-name` | `""`         | Name of the Kubernetes secret to store Tailscale state |
| -               | `INSECURE`           | `--insecure`    | `false`      | Allow insecure connection to the Kubernetes API        |

More options can be found in [values.yaml](helm/values.yaml).

## 🔗 Resources

- [Blog Post: Kubernetes API access over Tailscale](https://0x2321.de/kubernetes-api-access-over-tailscale/)
- [Tailscale Documentation](https://tailscale.com/kb/)
