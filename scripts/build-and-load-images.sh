#!/usr/bin/env bash

# Script para construir todas las imágenes Docker y cargarlas en Kind
set -euo pipefail

# Permite sobreescribir por variable de entorno
CLUSTER_NAME=${CLUSTER_NAME:-mybookstore}
IMAGE_TAG=${IMAGE_TAG:-latest}

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)

echo "🏗️  Construyendo imágenes Docker para MyBookStore..."

# Lista de servicios
BACKEND_SERVICES=("user" "catalog" "inventory" "cart" "order" "payment")
FRONTEND_SERVICES=("user" "catalog" "inventory" "cart" "order" "payment")

build_backend_image() {
    local service="$1"
    local df="$ROOT_DIR/Backend/src/${service}/Dockerfile"
    local tag="mybookstore-${service}-backend:${IMAGE_TAG}"
    echo "🔨 Construyendo imagen backend: $service (tag: $tag)"
    docker build -t "$tag" -f "$df" "$ROOT_DIR"
    echo "📦 Cargando imagen $tag en Kind ($CLUSTER_NAME)..."
    kind load docker-image "$tag" --name "$CLUSTER_NAME"
}

build_frontend_image() {
    local service="$1"
    local df="$ROOT_DIR/Frontend/src/${service}/Dockerfile"
    local tag="mybookstore-${service}-frontend:${IMAGE_TAG}"
    echo "🔨 Construyendo imagen frontend: $service (tag: $tag)"
    docker build -t "$tag" -f "$df" "$ROOT_DIR"
    echo "📦 Cargando imagen $tag en Kind ($CLUSTER_NAME)..."
    kind load docker-image "$tag" --name "$CLUSTER_NAME"
}

# Verificar que el cluster existe
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "❌ Cluster Kind '${CLUSTER_NAME}' no encontrado. Créalo primero con:"
    echo "   kind create cluster --config ${ROOT_DIR}/kind-config.yaml"
    exit 1
fi

echo "✅ Cluster Kind '${CLUSTER_NAME}' encontrado"

echo "🏗️  Construyendo servicios backend..."
for service in "${BACKEND_SERVICES[@]}"; do
    build_backend_image "$service"
done

echo "🏗️  Construyendo servicios frontend..."
for service in "${FRONTEND_SERVICES[@]}"; do
    build_frontend_image "$service"
done

echo "✅ Todas las imágenes han sido construidas y cargadas en Kind!"
echo
echo "📋 Imágenes disponibles:"
for service in "${BACKEND_SERVICES[@]}"; do
    echo "   - mybookstore-${service}-backend:${IMAGE_TAG}"
done
for service in "${FRONTEND_SERVICES[@]}"; do
    echo "   - mybookstore-${service}-frontend:${IMAGE_TAG}"
done

echo
echo "🎯 Si ya aplicaste los manifiestos, reinicia los deployments para que tomen las imágenes:"
echo "   kubectl rollout restart deploy -n mybookstore -l tier=backend"
echo "   kubectl rollout restart deploy -n mybookstore -l tier=frontend"