set -euo pipefail

command -v protoc >/dev/null || { echo "Falta protoc"; exit 1; }
command -v protoc-gen-go >/dev/null || { echo "Falta protoc-gen-go (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)"; exit 1; }
command -v protoc-gen-go-grpc >/dev/null || { echo "Falta protoc-gen-go-grpc (go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest)"; exit 1; }
command -v python3 >/dev/null || { echo "Falta python3"; exit 1; }
python3 -c "import grpc_tools.protoc" 2>/dev/null || { echo "Falta grpcio-tools (pip install grpcio-tools)"; exit 1; }

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$ROOT_DIR/proto"
OUT_DIR="$PROTO_DIR/gen"

mkdir -p "$OUT_DIR"

# Generar stubs de Go
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

# Generar stubs de Python en las carpetas correctas
python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/common" \
  --grpc_python_out="$OUT_DIR/common" \
  "$PROTO_DIR"/common.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/events" \
  --grpc_python_out="$OUT_DIR/events" \
  "$PROTO_DIR"/events.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/catalog" \
  --grpc_python_out="$OUT_DIR/catalog" \
  "$PROTO_DIR"/catalog.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/user" \
  --grpc_python_out="$OUT_DIR/user" \
  "$PROTO_DIR"/user.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/cart" \
  --grpc_python_out="$OUT_DIR/cart" \
  "$PROTO_DIR"/cart.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/order" \
  --grpc_python_out="$OUT_DIR/order" \
  "$PROTO_DIR"/order.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/inventory" \
  --grpc_python_out="$OUT_DIR/inventory" \
  "$PROTO_DIR"/inventory.proto

python3 -m grpc_tools.protoc -I "$PROTO_DIR" \
  --python_out="$OUT_DIR/payment" \
  --grpc_python_out="$OUT_DIR/payment" \
  "$PROTO_DIR"/payment.proto

# Crear archivos __init__.py para hacer que las carpetas sean módulos Python
echo "# Generated protobuf modules" > "$OUT_DIR/__init__.py"
echo "# Common protobuf modules" > "$OUT_DIR/common/__init__.py"
echo "# Events protobuf modules" > "$OUT_DIR/events/__init__.py"
echo "# Catalog protobuf modules" > "$OUT_DIR/catalog/__init__.py"
echo "# User protobuf modules" > "$OUT_DIR/user/__init__.py"
echo "# Cart protobuf modules" > "$OUT_DIR/cart/__init__.py"
echo "# Order protobuf modules" > "$OUT_DIR/order/__init__.py"
echo "# Inventory protobuf modules" > "$OUT_DIR/inventory/__init__.py"
echo "# Payment protobuf modules" > "$OUT_DIR/payment/__init__.py"

# Corregir imports en archivos Python para usar imports directos (sin módulos anidados)
find "$OUT_DIR" -name "*_pb2.py" -o -name "*_pb2_grpc.py" | while read file; do
    # Revertir cualquier import complejo a imports simples
    sed -i 's/from common import common_pb2/import common_pb2/g' "$file"
    sed -i 's/from events import events_pb2/import events_pb2/g' "$file"
    sed -i 's/from \. import \([a-z_]*\)_pb2/import \1_pb2/g' "$file"
done

echo "Archivos protobuf corregidos para imports directos"

echo "Stubs de Go y Python generados en $OUT_DIR" 