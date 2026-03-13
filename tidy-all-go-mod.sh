#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SERVICES=(
  "src/currencyservice"
  "src/frontend"
  "src/productcatalogservice"
  "src/shippingservice"
  "src/paymentservice"
  "src/adservice"
  "src/cartservice"
  "src/checkoutservice"
  "infra"
)

for service in "${SERVICES[@]}"; do
  dir="$SCRIPT_DIR/$service"

  if [ -f "$dir/go.mod" ]; then
    echo "=> running go mod tidy in $service"
    (cd "$dir" && go mod tidy)
    echo "ok: $service"
  else
    echo "skip: $service (go.mod not found)"
  fi
done

echo "All listed services have been processed."