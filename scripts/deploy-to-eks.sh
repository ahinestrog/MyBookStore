echo "🚀 Desplegando MyBookStore en EKS..."

# Colores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() {
  echo ""
  echo -e "${BLUE}==== $1 ====${NC}"
  echo ""
}

print_success() {
  echo -e "${GREEN}✅ $1${NC}"
}

# Verificar que kubectl está conectado al cluster correcto
print_step "1. Verificando conexión al cluster"
kubectl cluster-info
if [ $? -ne 0 ]; then
  echo "❌ No se pudo conectar al cluster. Verifica que eksctl haya configurado kubectl correctamente."
  exit 1
fi
print_success "Conectado al cluster EKS"

# Crear namespace
print_step "2. Creando namespace"
kubectl apply -f k8s/00-namespace.yaml
print_success "Namespace creado"

# Esperar un momento
sleep 2

# Aplicar ConfigMap
print_step "3. Aplicando ConfigMap"
kubectl apply -f k8s/01-configmap.yaml
kubectl get configmap -n mybookstore
print_success "ConfigMap aplicado"

# Desplegar RabbitMQ
print_step "4. Desplegando RabbitMQ"
kubectl apply -f k8s/03-rabbitmq/
echo "Esperando a que RabbitMQ esté listo..."
kubectl wait --for=condition=ready pod -l app=rabbitmq -n mybookstore --timeout=180s
if [ $? -eq 0 ]; then
  print_success "RabbitMQ está corriendo"
else
  echo "⚠️  RabbitMQ tardó más de lo esperado. Verifica los logs:"
  echo "  kubectl logs -n mybookstore -l app=rabbitmq"
fi

# Desplegar Backends
print_step "5. Desplegando Backends"
kubectl apply -f k8s/04-backends/
echo "Esperando 15 segundos para que los pods inicien..."
sleep 15
kubectl get pods -n mybookstore -l tier=backend
print_success "Backends desplegados"

# Desplegar Frontends
print_step "6. Desplegando Frontends"
kubectl apply -f k8s/05-frontends/
echo "Esperando 15 segundos para que los pods inicien..."
sleep 15
kubectl get pods -n mybookstore -l tier=frontend
print_success "Frontends desplegados"

# Desplegar Ingress
print_step "7. Desplegando Ingress"
kubectl apply -f k8s/06-ingress.yaml
kubectl get ingress -n mybookstore
print_success "Ingress desplegado"

# Resumen
print_step "RESUMEN DEL DESPLIEGUE"

echo "📊 Estado de los Pods:"
kubectl get pods -n mybookstore

echo ""
echo "🌐 Servicios:"
kubectl get svc -n mybookstore

echo ""
echo "🔗 Ingress:"
kubectl get ingress -n mybookstore

echo ""
echo "🌍 URL de tu aplicación:"
export LB_HOSTNAME=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

if [ -z "$LB_HOSTNAME" ]; then
  echo "⚠️  LoadBalancer aún no tiene hostname asignado."
  echo "   Espera 1-2 minutos y ejecuta:"
  echo "   kubectl get svc -n ingress-nginx ingress-nginx-controller"
else
  echo ""
  echo "Catalog:   http://${LB_HOSTNAME}/catalog/"
  echo "User:      http://${LB_HOSTNAME}/user/"
  echo "Cart:      http://${LB_HOSTNAME}/cart/"
  echo "Inventory: http://${LB_HOSTNAME}/inventory/"
  echo "Order:     http://${LB_HOSTNAME}/order/"
  echo "Payment:   http://${LB_HOSTNAME}/payment/"
  echo "RabbitMQ:  http://${LB_HOSTNAME}/rabbitmq/"
  echo ""
fi

print_success "¡Despliegue completado!"

echo ""
echo "📝 Comandos útiles:"
echo "  Ver logs de un servicio:"
echo "    kubectl logs -n mybookstore -l app=catalog,tier=backend"
echo ""
echo "  Ver todos los pods:"
echo "    kubectl get pods -n mybookstore"
echo ""
echo "  Ver eventos:"
echo "    kubectl get events -n mybookstore --sort-by='.lastTimestamp'"
echo ""
echo "  Reiniciar un deployment:"
echo "    kubectl rollout restart deployment/catalog-backend -n mybookstore"
echo ""
