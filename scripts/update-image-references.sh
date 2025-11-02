export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export AWS_REGION=us-east-1
export ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo "🔧 Actualizando referencias de imágenes en deployments..."
echo "Registry: ${ECR_REGISTRY}"

# Función para actualizar deployment
update_deployment() {
  local file=$1
  local old_image=$2
  local new_image="${ECR_REGISTRY}/${old_image}"

  sed -i.bak "s|image: ${old_image}:latest|image: ${new_image}:latest|g" ${file}
  
  if [ $? -eq 0 ]; then
    echo "✅ Actualizado: ${file}"
    rm -f ${file}.bak
  else
    echo "❌ Error actualizando: ${file}"
  fi
}

# Backends
echo ""
echo "=== BACKENDS ==="
update_deployment "k8s/04-backends/user-deployment.yaml" "mybookstore-user-backend"
update_deployment "k8s/04-backends/catalog-deployment.yaml" "mybookstore-catalog-backend"
update_deployment "k8s/04-backends/cart-deployment.yaml" "mybookstore-cart-backend"
update_deployment "k8s/04-backends/inventory-deployment.yaml" "mybookstore-inventory-backend"
update_deployment "k8s/04-backends/order-deployment.yaml" "mybookstore-order-backend"
update_deployment "k8s/04-backends/payment-deployment.yaml" "mybookstore-payment-backend"

# Frontends
echo ""
echo "=== FRONTENDS ==="
update_deployment "k8s/05-frontends/frontend-user-deployment.yaml" "mybookstore-user-frontend"
update_deployment "k8s/05-frontends/frontend-catalog-deployment.yaml" "mybookstore-catalog-frontend"
update_deployment "k8s/05-frontends/frontend-cart-deployment.yaml" "mybookstore-cart-frontend"
update_deployment "k8s/05-frontends/frontend-inventory-deployment.yaml" "mybookstore-inventory-frontend"
update_deployment "k8s/05-frontends/frontend-order-deployment.yaml" "mybookstore-order-frontend"
update_deployment "k8s/05-frontends/frontend-payment-deployment.yaml" "mybookstore-payment-frontend"

echo ""
echo "🎉 ¡Referencias de imágenes actualizadas!"
echo ""
echo "Siguiente paso:"
echo "  kubectl apply -f k8s/00-namespace.yaml"
echo "  kubectl apply -f k8s/01-configmap.yaml"
echo "  kubectl apply -f k8s/03-rabbitmq/"
echo "  kubectl apply -f k8s/04-backends/"
echo "  kubectl apply -f k8s/05-frontends/"
echo "  kubectl apply -f k8s/06-ingress.yaml"
