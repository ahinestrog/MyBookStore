echo "🧹 Limpiando recursos de AWS..."
echo ""
echo "⚠️  ADVERTENCIA: Esto eliminará TODOS los recursos de MyBookStore en AWS"
echo ""
read -p "¿Estás seguro? (escribe 'si' para confirmar): " confirm

if [ "$confirm" != "si" ]; then
  echo "Operación cancelada"
  exit 0
fi

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

print_step() {
  echo ""
  echo -e "${GREEN}==== $1 ====${NC}"
}

# 1. Eliminar recursos de Kubernetes
print_step "1. Eliminando recursos de Kubernetes"
kubectl delete namespace mybookstore --ignore-not-found=true
kubectl delete namespace ingress-nginx --ignore-not-found=true
echo "✅ Namespaces eliminados"

# 2. Eliminar cluster EKS
print_step "2. Eliminando cluster EKS (esto puede tomar 10-15 minutos)"
eksctl delete cluster --name mybookstore-cluster --region us-east-1
if [ $? -eq 0 ]; then
  echo "✅ Cluster EKS eliminado"
else
  echo "⚠️  Error eliminando cluster. Puede que no exista o ya esté eliminado."
fi

# 3. Eliminar repositorios ECR
print_step "3. Eliminando repositorios ECR"
SERVICES=(
  "user-backend"
  "catalog-backend"
  "cart-backend"
  "inventory-backend"
  "order-backend"
  "payment-backend"
  "user-frontend"
  "catalog-frontend"
  "cart-frontend"
  "inventory-frontend"
  "order-frontend"
  "payment-frontend"
)

for service in "${SERVICES[@]}"; do
  echo "Eliminando: mybookstore-${service}..."
  aws ecr delete-repository \
    --repository-name mybookstore-${service} \
    --region us-east-1 \
    --force \
    2>/dev/null
  
  if [ $? -eq 0 ]; then
    echo "  ✅ mybookstore-${service}"
  else
    echo "  ⚠️  mybookstore-${service} (puede que no exista)"
  fi
done

print_step "LIMPIEZA COMPLETADA"
echo ""
echo "✅ Todos los recursos han sido eliminados"
echo ""
echo "💰 Verifica en AWS Console que no queden recursos:"
echo "  - EKS: https://console.aws.amazon.com/eks/"
echo "  - EC2: https://console.aws.amazon.com/ec2/"
echo "  - ECR: https://console.aws.amazon.com/ecr/"
echo "  - VPC: https://console.aws.amazon.com/vpc/"
echo ""
echo "  Billing: https://console.aws.amazon.com/billing/"
echo ""
