#!/bin/bash

# Script para construir todas las imágenes Docker y cargarlas en Kind
set -e

CLUSTER_NAME="kind"
ROOT_DIR="/mnt/c/Users/$(whoami)/Ubuntu/Topicos\ especiales\ en\ Telematica/MyBookStore"
IMAGE_TAG="latest"

echo "🏗️  Construyendo imágenes Docker para MyBookStore..."

# Lista de servicios backend
BACKEND_SERVICES=("user" "catalog" "inventory" "cart" "order" "payment")

# Lista de servicios frontend  
FRONTEND_SERVICES=("user" "catalog" "inventory" "cart" "order" "payment")

# Función para construir imagen backend
build_backend_image() {
    local service=$1
    echo "🔨 Construyendo imagen backend: $service"
    
    docker build -t mybookstore-$service-backend:$IMAGE_TAG \
        -f Backend/src/$service/Dockerfile \
        .
    
    echo "📦 Cargando imagen mybookstore-$service-backend:$IMAGE_TAG en Kind..."
    kind load docker-image mybookstore-$service-backend:$IMAGE_TAG --name $CLUSTER_NAME
}

# Función para construir imagen frontend
build_frontend_image() {
    local service=$1
    echo "🔨 Construyendo imagen frontend: $service"
    
    docker build -t mybookstore-$service-frontend:$IMAGE_TAG \
        -f Frontend/src/$service/Dockerfile \
        .
    
    echo "📦 Cargando imagen mybookstore-$service-frontend:$IMAGE_TAG en Kind..."
    kind load docker-image mybookstore-$service-frontend:$IMAGE_TAG --name $CLUSTER_NAME
}

# Verificar que Kind esté corriendo
if ! kind get clusters | grep -q "^$CLUSTER_NAME$"; then
    echo "❌ Cluster Kind '$CLUSTER_NAME' no encontrado. Créalo primero con:"
    echo "   kind create cluster --name $CLUSTER_NAME"
    exit 1
fi

echo "✅ Cluster Kind '$CLUSTER_NAME' encontrado"

# Construir imágenes backend
echo "🏗️  Construyendo servicios backend..."
for service in "${BACKEND_SERVICES[@]}"; do
    build_backend_image $service
done

# Construir imágenes frontend
echo "🏗️  Construyendo servicios frontend..."
for service in "${FRONTEND_SERVICES[@]}"; do
    build_frontend_image $service
done

echo "✅ Todas las imágenes han sido construidas y cargadas en Kind!"
echo ""
echo "📋 Imágenes disponibles:"
for service in "${BACKEND_SERVICES[@]}"; do
    echo "   - mybookstore-$service-backend:$IMAGE_TAG"
done
for service in "${FRONTEND_SERVICES[@]}"; do
    echo "   - mybookstore-$service-frontend:$IMAGE_TAG"
done

echo ""
echo "🎯 Siguiente paso: kubectl apply -f k8s/"