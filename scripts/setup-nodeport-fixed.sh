#!/bin/bash
set -e

echo "=== 1. Instalando Ingress NGINX ==="
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.1/deploy/static/provider/aws/deploy.yaml

echo "Esperando a que el controller esté Ready..."
kubectl -n ingress-nginx wait \
  --for=condition=ready pod \
  -l app.kubernetes.io/component=controller \
  --timeout=600s

echo ""
echo "=== 2. Cambiando servicio a NodePort ==="
kubectl patch svc ingress-nginx-controller -n ingress-nginx -p '{"spec":{"type":"NodePort"}}'

echo ""
echo "=== 3. Obteniendo NodePort ==="
HTTP_NODEPORT=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}')
echo "HTTP NodePort: ${HTTP_NODEPORT}"

echo ""
echo "=== 4. Buscando Security Group de los nodos ==="

# Método 1: Por tag de cluster
NODE_SG=$(aws ec2 describe-security-groups \
  --region us-east-1 \
  --filters "Name=tag:eks:cluster-name,Values=mybookstore-cluster" \
            "Name=tag:aws:eks:cluster-name,Values=mybookstore-cluster" \
  --query 'SecurityGroups[?contains(GroupName, `node`)].GroupId' \
  --output text | head -1)

# Método 2: Por instancias EC2 si el método 1 falla
if [ -z "$NODE_SG" ]; then
  echo "Método 1 falló, intentando método 2..."
  
  INSTANCE_ID=$(aws ec2 describe-instances \
    --region us-east-1 \
    --filters "Name=tag:eks:cluster-name,Values=mybookstore-cluster" \
              "Name=instance-state-name,Values=running" \
    --query 'Reservations[0].Instances[0].InstanceId' \
    --output text)
  
  echo "Instance ID encontrada: ${INSTANCE_ID}"
  
  NODE_SG=$(aws ec2 describe-instances \
    --region us-east-1 \
    --instance-ids ${INSTANCE_ID} \
    --query 'Reservations[0].Instances[0].SecurityGroups[0].GroupId' \
    --output text)
fi

if [ -z "$NODE_SG" ] || [ "$NODE_SG" == "None" ]; then
  echo "❌ No se pudo encontrar el Security Group automáticamente"
  echo "Búscalo manualmente en AWS Console → EC2 → Instances"
  read -p "Ingresa el Security Group ID: " NODE_SG
fi

echo "Security Group: ${NODE_SG}"

echo ""
echo "=== 5. Abriendo puerto ${HTTP_NODEPORT} en Security Group ==="
aws ec2 authorize-security-group-ingress \
  --region us-east-1 \
  --group-id ${NODE_SG} \
  --protocol tcp \
  --port ${HTTP_NODEPORT} \
  --cidr 0.0.0.0/0 2>&1 | grep -v "already exists" || echo "✅ Puerto abierto"

echo ""
echo "=== 6. Obteniendo IP pública de los nodos ==="
NODE_IP=$(kubectl get nodes -o wide | awk '/ Ready / && $2 !~ /SchedulingDisabled/ {print $7; exit}')

if [ -z "$NODE_IP" ] || [ "$NODE_IP" == "<none>" ]; then
  echo "❌ Los nodos no tienen IP pública"
  echo "Verifica que creaste el nodegroup con --node-private-networking=false"
  exit 1
fi

echo ""
echo "=== 7. URLs de acceso ==="
echo ""
echo "Node IP: ${NODE_IP}"
echo "NodePort: ${HTTP_NODEPORT}"
echo ""
echo "🎉 Aplicación disponible en:"
echo ""
echo "Catalog:   http://${NODE_IP}:${HTTP_NODEPORT}/catalog/"
echo "User:      http://${NODE_IP}:${HTTP_NODEPORT}/user/"
echo "Cart:      http://${NODE_IP}:${HTTP_NODEPORT}/cart/"
echo "Inventory: http://${NODE_IP}:${HTTP_NODEPORT}/inventory/"
echo "Order:     http://${NODE_IP}:${HTTP_NODEPORT}/order/"
echo "Payment:   http://${NODE_IP}:${HTTP_NODEPORT}/payment/"
echo ""

echo "=== 8. Probando conexión ==="
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://${NODE_IP}:${HTTP_NODEPORT}/" 2>/dev/null || echo "000")
echo "HTTP Status Code: ${HTTP_CODE}"

if echo "${HTTP_CODE}" | grep -q "200\|301\|302\|404"; then
  echo "✅ Ingress accesible (${HTTP_CODE})!"
  if [ "${HTTP_CODE}" == "404" ]; then
    echo "   (404 es normal, aún no has desplegado la app)"
  fi
else
  echo "⚠️  No accesible (${HTTP_CODE}). Verifica SG y espera 1-2 min."
fi