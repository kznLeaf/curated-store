#!/bin/bash

# 1. 创建命名空间（如果不存在）
kubectl get ns dex >/dev/null 2>&1 || kubectl create namespace dex

echo "--- Preparing TLS Secret ---"
kubectl -n dex create secret tls dex.example.com.tls \
  --cert=./kubernetes-manifests/components/Dex/ssl/cert.pem \
  --key=./kubernetes-manifests/components/Dex/ssl/key.pem \

echo "--- Preparing GitHub Credentials ---"
if [ -z "$GITHUB_CLIENT_ID" ] || [ -z "$GITHUB_CLIENT_SECRET" ]; then
    echo "Error: GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET is not set in your environment."
    exit 1
fi

kubectl -n dex create secret generic github-client \
  --from-literal=client-id="$GITHUB_CLIENT_ID" \
  --from-literal=client-secret="$GITHUB_CLIENT_SECRET" \

echo "--- All secrets are ready in namespace 'dex' ---"

