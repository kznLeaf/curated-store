#!/bin/bash -eu
# 执行所有的 genproto.sh 脚本，刷新所有服务的 .proto 生成代码
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

SERVICES=(
  "src/currencyservice"
  "src/frontend"
  "src/productcatalogservice"
  "src/shippingservice"
  "src/paymentservice"
)

for service in "${SERVICES[@]}"; do
  dir="$SCRIPT_DIR/$service"
  if [ -f "$dir/genproto.sh" ]; then
    echo "▶ 正在处理 $service ..."
    (cd "$dir" && bash genproto.sh)
    echo "✓ $service 完成"
  else
    echo "⚠ 跳过 $service：未找到 genproto.sh"
  fi
done

echo ""
echo ".proto全部更新完毕"