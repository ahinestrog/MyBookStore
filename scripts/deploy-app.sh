#!/bin/bash
set -e

cd /home/alejo/dev/topicosTelematica/MyBookStore

echo "=== 1. Configurando ECR ==="
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo "ECR Registry: ${ECR_REGISTRY}"

echo ""
echo "=== 2. Login a ECR ==="
aws ecr get-login-password --region $AWS_REGION \
| docker login --username AWS --password-stdin $ECR_REGISTRY

echo ""
echo "=== 3. Creando repositorios ECR ==="
./scripts/create-ecr-repos.sh

echo ""
echo "=== 4. Construyendo y subiendo imágenes (tarda ~15 min) ==="
./scripts/build-and-push.sh

echo ""
echo "=== 5. Actualizando manifests ==="
./scripts/update-image-references.sh

echo ""
echo "=== 6. Desplegando aplicación ==="
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/01-configmap.yaml
kubectl apply -f k8s/03-rabbitmq/

echo "Esperando RabbitMQ..."
kubectl wait -n mybookstore --for=condition=ready pod -l app=rabbitmq --timeout=300s

kubectl apply -f k8s/04-backends/
kubectl apply -f k8s/05-frontends/
kubectl apply -f k8s/06-ingress.yaml

echo ""
echo "=== 7. Esperando que los pods estén Ready ==="
sleep 20
kubectl get pods -n mybookstore

echo ""
echo "=== 8. Verificando Ingress ==="
kubectl get ingress -n mybookstore

echo ""
echo "=== 9. URLs de acceso ==="
HTTP_NODEPORT=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}')
NODE_IP=$(kubectl get nodes -o wide | grep -v SchedulingDisabled | grep Ready | head -1 | awk '{print $7}')

echo ""
echo "🎉 Aplicación disponible en:"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/catalog/"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/user/"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/cart/"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/inventory/"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/order/"
echo "http://${NODE_IP}:${HTTP_NODEPORT}/payment/"
echo ""