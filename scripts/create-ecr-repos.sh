# Configuración
export AWS_REGION=us-east-1

echo "📦 Creando repositorios en ECR..."

# Lista de servicios
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

# Crear repositorio para cada servicio
for service in "${SERVICES[@]}"; do
  echo "Creando repositorio: mybookstore-${service}..."
  
  aws ecr create-repository \
    --repository-name mybookstore-${service} \
    --region ${AWS_REGION} \
    --image-scanning-configuration scanOnPush=true \
    2>/dev/null
  
  if [ $? -eq 0 ]; then
    echo "✅ Creado: mybookstore-${service}"
  else
    echo "⚠️  mybookstore-${service} (puede que ya exista)"
  fi
done

# Obtener URL del registry
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo ""
echo "🎉 ¡Repositorios creados!"
echo ""
echo "Tu ECR Registry: ${ECR_REGISTRY}"
echo ""
echo "Siguiente paso: Autenticar Docker con ECR"
echo "  aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${ECR_REGISTRY}"
