# Configuración
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export AWS_REGION=us-east-1
export ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo "🚀 Construyendo y subiendo imágenes a ECR..."
echo "Registry: ${ECR_REGISTRY}"

# Función para build y push
build_and_push() {
  local service=$1
  local type=$2
  local dockerfile=$3
  local context=$4
  
  echo ""
  echo "📦 Procesando: mybookstore-${service}-${type}"
  
  # Build
  docker build -t mybookstore-${service}-${type}:latest \
    -f ${dockerfile} ${context}
  
  if [ $? -ne 0 ]; then
    echo "❌ Error building mybookstore-${service}-${type}"
    return 1
  fi
  
  # Tag para ECR
  docker tag mybookstore-${service}-${type}:latest \
    ${ECR_REGISTRY}/mybookstore-${service}-${type}:latest
  
  # Push a ECR
  docker push ${ECR_REGISTRY}/mybookstore-${service}-${type}:latest
  
  if [ $? -ne 0 ]; then
    echo "❌ Error pushing mybookstore-${service}-${type}"
    return 1
  fi
  
  echo "✅ mybookstore-${service}-${type} subido exitosamente"
}

# Backend services
echo "=== BACKENDS ==="
build_and_push "user" "backend" "Backend/src/user/Dockerfile" "Backend/src/user"
build_and_push "catalog" "backend" "Backend/src/catalog/Dockerfile" "Backend/src/catalog"
build_and_push "cart" "backend" "Backend/src/cart/Dockerfile" "Backend/src/cart"
build_and_push "inventory" "backend" "Backend/src/inventory/Dockerfile" "Backend/src/inventory"
build_and_push "order" "backend" "Backend/src/order/Dockerfile" "Backend/src/order"
build_and_push "payment" "backend" "Backend/src/payment/Dockerfile" "Backend/src/payment"

# Frontend services
echo ""
echo "=== FRONTENDS ==="
build_and_push "user" "frontend" "Frontend/src/user/Dockerfile" "Frontend/src/user"
build_and_push "catalog" "frontend" "Frontend/src/catalog/Dockerfile" "Frontend/src/catalog"
build_and_push "cart" "frontend" "Frontend/src/cart/Dockerfile" "Frontend/src/cart"
build_and_push "inventory" "frontend" "Frontend/src/inventory/Dockerfile" "Frontend/src/inventory"
build_and_push "order" "frontend" "Frontend/src/order/Dockerfile" "Frontend/src/order"
build_and_push "payment" "frontend" "Frontend/src/payment/Dockerfile" "Frontend/src/payment"

echo ""
echo "🎉 ¡Todas las imágenes han sido construidas y subidas a ECR!"
echo ""
echo "Siguiente paso:"
echo "  ./update-image-references.sh"
