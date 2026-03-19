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

Then, run `./set-up-secrets.sh` before you deploy resources, which includes two steps:

1. Create TLS Secret in `default` namespace:

```bash
kubectl -n default create secret tls dex.example.com.tls \
  --cert=./cert.pem \
  --key=./key.pem
```

The CA file will be mounted as Secret in `set-up-secrets.sh` so that the frontendservice can access it.

2. Create GitHub Credentials Secret in `default` namespace, and before that you must ensure `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` have been added as your shell environment varibles.

```bash
kubectl -n default create secret generic github-client \
  --from-literal=client-id=$GITHUB_CLIENT_ID \
  --from-literal=client-secret=$GITHUB_CLIENT_SECRET
```
