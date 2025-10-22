set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$ROOT_DIR/proto"
OUT_DIR="$PROTO_DIR/gen"

mkdir -p "$OUT_DIR"

protoc -I "$PROTO_DIR" \
  --go_out="$OUT_DIR" --go_opt=paths=source_relative \
  --go-grpc_out="$OUT_DIR" --go-grpc_opt=paths=source_relative \
  "$PROTO_DIR"/common.proto \
  "$PROTO_DIR"/catalog.proto \
  "$PROTO_DIR"/user.proto \
  "$PROTO_DIR"/cart.proto \
  "$PROTO_DIR"/order.proto \
  "$PROTO_DIR"/inventory.proto \
  "$PROTO_DIR"/payment.proto \
  "$PROTO_DIR"/events.proto

echo "Stubs generados en $OUT_DIR" 