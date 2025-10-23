set -euo pipefail

command -v protoc >/dev/null || { echo "Falta protoc"; exit 1; }
command -v protoc-gen-go >/dev/null || { echo "Falta protoc-gen-go (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)"; exit 1; }
command -v protoc-gen-go-grpc >/dev/null || { echo "Falta protoc-gen-go-grpc (go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest)"; exit 1; }

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$ROOT_DIR/proto"
OUT_DIR="$PROTO_DIR/gen"

mkdir -p "$OUT_DIR"

protoc -I "$PROTO_DIR" \
  --go_out="$OUT_DIR" --go_opt=module=github.com/ahinestrog/mybookstore/proto/gen \
  --go-grpc_out="$OUT_DIR" --go-grpc_opt=module=github.com/ahinestrog/mybookstore/proto/gen \
  "$PROTO_DIR"/common.proto \
  "$PROTO_DIR"/catalog.proto \
  "$PROTO_DIR"/user.proto \
  "$PROTO_DIR"/cart.proto \
  "$PROTO_DIR"/order.proto \
  "$PROTO_DIR"/inventory.proto \
  "$PROTO_DIR"/payment.proto \
  "$PROTO_DIR"/events.proto

echo "Stubs generados en $OUT_DIR" 