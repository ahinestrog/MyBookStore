echo "🔍 Verificando el despliegue de MyBookStore en AWS EKS..."
echo ""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Contadores
PASS=0
FAIL=0

check_pass() {
  echo -e "${GREEN}✅ $1${NC}"
  ((PASS++))
}

check_fail() {
  echo -e "${RED}❌ $1${NC}"
  ((FAIL++))
}

check_warn() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

# 1. Verificar que kubectl está conectado
echo "1️⃣  Verificando conexión a cluster..."
kubectl cluster-info > /dev/null 2>&1
if [ $? -eq 0 ]; then
  check_pass "Conectado a cluster Kubernetes"
else
  check_fail "No hay conexión al cluster"
  exit 1
fi

# 2. Verificar namespace
echo ""
echo "2️⃣  Verificando namespace..."
kubectl get namespace mybookstore > /dev/null 2>&1
if [ $? -eq 0 ]; then
  check_pass "Namespace 'mybookstore' existe"
else
  check_fail "Namespace 'mybookstore' no existe"
fi

# 3. Verificar ConfigMap
echo ""
echo "3️⃣  Verificando ConfigMap..."
kubectl get configmap mybookstore-config -n mybookstore > /dev/null 2>&1
if [ $? -eq 0 ]; then
  check_pass "ConfigMap configurado"
else
  check_fail "ConfigMap no encontrado"
fi

# 4. Verificar RabbitMQ
echo ""
echo "4️⃣  Verificando RabbitMQ..."
RABBITMQ_READY=$(kubectl get pods -n mybookstore -l app=rabbitmq -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null)
if [ "$RABBITMQ_READY" = "true" ]; then
  check_pass "RabbitMQ está corriendo"
else
  check_fail "RabbitMQ no está listo"
fi

# 5. Verificar Backends
echo ""
echo "5️⃣  Verificando Backends..."
BACKENDS=("user" "catalog" "cart" "inventory" "order" "payment")
for backend in "${BACKENDS[@]}"; do
  READY=$(kubectl get pods -n mybookstore -l app=${backend},tier=backend -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null)
  if [ "$READY" = "true" ]; then
    check_pass "${backend}-backend está corriendo"
  else
    check_fail "${backend}-backend no está listo"
  fi
done

# 6. Verificar Frontends
echo ""
echo "6️⃣  Verificando Frontends..."
FRONTENDS=("user" "catalog" "cart" "inventory" "order" "payment")
for frontend in "${FRONTENDS[@]}"; do
  READY=$(kubectl get pods -n mybookstore -l app=${frontend},tier=frontend -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null)
  if [ "$READY" = "true" ]; then
    check_pass "${frontend}-frontend está corriendo"
  else
    check_fail "${frontend}-frontend no está listo"
  fi
done

# 7. Verificar Servicios
echo ""
echo "7️⃣  Verificando Servicios..."
SERVICES=$(kubectl get svc -n mybookstore --no-headers | wc -l)
if [ $SERVICES -ge 13 ]; then
  check_pass "Todos los servicios están creados ($SERVICES servicios)"
else
  check_warn "Solo hay $SERVICES servicios (deberían ser al menos 13)"
fi

# 8. Verificar Ingress
echo ""
echo "8️⃣  Verificando Ingress..."
kubectl get ingress mybookstore-ingress -n mybookstore > /dev/null 2>&1
if [ $? -eq 0 ]; then
  check_pass "Ingress configurado"
else
  check_fail "Ingress no encontrado"
fi

# 9. Verificar Nginx Ingress Controller
echo ""
echo "9️⃣  Verificando Nginx Ingress Controller..."
kubectl get pods -n ingress-nginx > /dev/null 2>&1
if [ $? -eq 0 ]; then
  NGINX_READY=$(kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null)
  if [ "$NGINX_READY" = "true" ]; then
    check_pass "Nginx Ingress Controller está corriendo"
  else
    check_fail "Nginx Ingress Controller no está listo"
  fi
else
  check_fail "Nginx Ingress Controller no está instalado"
fi

# 10. Verificar LoadBalancer
echo ""
echo "🔟 Verificando LoadBalancer..."
LB_HOSTNAME=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null)
if [ -n "$LB_HOSTNAME" ]; then
  check_pass "LoadBalancer tiene hostname asignado"
  echo ""
  echo "   🌍 URL de tu aplicación:"
  echo "   http://${LB_HOSTNAME}/catalog/"
else
  check_warn "LoadBalancer aún no tiene hostname (puede tomar 1-2 minutos)"
fi

echo ""
echo "════════════════════════════════════════"
echo "RESUMEN DE VERIFICACIÓN"
echo "════════════════════════════════════════"
echo -e "${GREEN}Exitosos: $PASS${NC}"
if [ $FAIL -gt 0 ]; then
  echo -e "${RED}Fallidos:  $FAIL${NC}"
fi
echo ""

if [ $FAIL -eq 0 ]; then
  echo -e "${GREEN}🎉 ¡Todo está funcionando correctamente!${NC}"
  echo ""
  if [ -n "$LB_HOSTNAME" ]; then
    echo "Puedes acceder a tu aplicación en:"
    echo "  http://${LB_HOSTNAME}/catalog/"
    echo ""
    echo "Prueba con curl:"
    echo "  curl -I http://${LB_HOSTNAME}/catalog/"
  fi
else
  echo -e "${RED}⚠️  Hay algunos problemas. Revisa los errores arriba.${NC}"
  echo ""
  echo "Comandos útiles para debugging:"
  echo "  kubectl get pods -n mybookstore"
  echo "  kubectl logs -n mybookstore -l app=<SERVICE-NAME>"
  echo "  kubectl describe pod -n mybookstore <POD-NAME>"
fi

echo ""
echo "📊 Información adicional:"
echo ""
echo "Pods en mybookstore:"
kubectl get pods -n mybookstore
echo ""
echo "Servicios en mybookstore:"
kubectl get svc -n mybookstore
echo ""
