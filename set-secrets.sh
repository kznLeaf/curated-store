#!/bin/bash

kubectl -n default delete secret oauth2-proxy-credentials --ignore-not-found

kubectl create secret generic oauth2-proxy-credentials \
  --from-literal=OAUTH2_PROXY_CLIENT_ID="$OAUTH2_PROXY_CLIENT_ID" \
  --from-literal=OAUTH2_PROXY_CLIENT_SECRET="$OAUTH2_PROXY_CLIENT_SECRET" \
  --from-literal=OAUTH2_PROXY_COOKIE_SECRET="$OAUTH2_PROXY_COOKIE_SECRET"

echo "Secrets for oauth2-proxy have been set successfully."