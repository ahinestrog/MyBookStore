#!/bin/bash
# Script para probar diferentes escenarios de inventario
# Cambia la disponibilidad sin reiniciar pods

set -e

BOOK_ID=${1:-1}
NEW_QTY=${2:-0}

echo "📦 Actualizando inventario para libro ID: $BOOK_ID → Cantidad: $NEW_QTY"

# Obtener el nombre del pod de inventario
POD=$(kubectl -n mybookstore get pods -l app=inventory -o jsonpath='{.items[0].metadata.name}')
echo "🔍 Pod encontrado: $POD"

# Copiar la base de datos del pod
echo "1️⃣ Descargando base de datos..."
kubectl -n mybookstore cp $POD:/data/inventory.db /tmp/inventory.db 2>/dev/null

# Verificar estado actual
echo "📊 Estado actual:"
sqlite3 /tmp/inventory.db "SELECT book_id, total_qty, reserved_qty, (total_qty - reserved_qty) as available FROM stock ORDER BY book_id;"

echo ""
echo "2️⃣ Actualizando libro ID $BOOK_ID a $NEW_QTY unidades..."
sqlite3 /tmp/inventory.db "UPDATE stock SET total_qty = $NEW_QTY WHERE book_id = $BOOK_ID;"

# Verificar el cambio
echo "✓ Verificando cambio:"
sqlite3 /tmp/inventory.db "SELECT book_id, total_qty, reserved_qty, (total_qty - reserved_qty) as available FROM stock WHERE book_id = $BOOK_ID;"

# Copiar de vuelta al pod
echo ""
echo "3️⃣ Subiendo base de datos actualizada..."
kubectl -n mybookstore cp /tmp/inventory.db $POD:/data/inventory.db

# Reiniciar el pod de inventario para que recargue
echo "4️⃣ Reiniciando servicio de inventario..."
kubectl -n mybookstore delete pod $POD

echo ""
echo "⏳ Esperando que el nuevo pod esté listo..."
kubectl -n mybookstore wait --for=condition=ready pod -l app=inventory --timeout=60s

echo ""
echo "🎉 ¡Listo! El libro ID $BOOK_ID ahora tiene $NEW_QTY unidades disponibles"
echo ""
echo "🧪 Puedes verificarlo con:"
echo "   curl -s 'http://localhost/catalog/book?id=$BOOK_ID' | grep Disponibilidad"
