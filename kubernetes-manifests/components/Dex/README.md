## Setup Guide

### Create GitHub OAuth App
Register a new OAuth application [here](https://github.com/settings/applications/new). 

![](./image.png)

Ensure the **Authorization callback URL** matches your Dex configuration (e.g., `https://localhost:5556/dex/callback`).

### Generate TLS Assets

Running Dex with HTTPS enabled requires a valid SSL certificate. Generate the necessary SSL certificates and CA files by running:

```bash
./src/frontend/gencert.sh
```

The assets will be generated in the `ssl/` directory.

### Run Frontend Service

Start the local frontend client using the generated CA:

```bash
go run . -issuer-root-ca ./ssl/ca.pem
```

### Configure Kubernetes Secrets

First, run `gencert.sh` to generate necessary files. 

The, run `./set-up-secrets.sh` before you deploy resources, which includes three steps:

1. Create Namespace `dex` for Dex:

```bash
kubectl create namespace dex
```

2. Create TLS Secret:

```bash
kubectl -n dex create secret tls dex.example.com.tls \
  --cert=./cert.pem \
  --key=./key.pem
```

3. Create GitHub Credentials Secret, and before that you must ensure `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` have been added as your shell environment varibles.

```bash
kubectl -n dex create secret generic github-client \
  --from-literal=client-id=$GITHUB_CLIENT_ID \
  --from-literal=client-secret=$GITHUB_CLIENT_SECRET
```
