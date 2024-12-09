#!/bin/bash

# Actualiza los paquetes e instala dependencias si es necesario
sudo apt-get update -y
sudo apt-get install -y apt-transport-https curl

# Descarga e instala Helm si no está instalado
if ! command -v helm &> /dev/null
then
    echo "Helm no está instalado. Instalando Helm..."
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
else
    echo "Helm ya está instalado."
fi

# Agrega el repositorio de Jetstack
echo "Agregando el repositorio de Jetstack..."
helm repo add jetstack https://charts.jetstack.io

# Actualiza los repositorios
echo "Actualizando repositorios de Helm..."
helm repo update

echo "Configuración completada con éxito."

echo "Instalando cert-manager..."

kubectl create namespace cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --set installCRDs=true

#kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml

echo "Cert-manager instalado con éxito."

git clone https://github.com/baarde/cert-manager-webhook-ovh.git
cd cert-manager-webhook-ovh
helm install cert-manager-webhook-ovh ./deploy/cert-manager-webhook-ovh --set groupName='n1rocket.com'