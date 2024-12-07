#!/bin/bash

# Salir inmediatamente si ocurre un error
set -e

# Variables
IMAGE_NAME="app"
IMAGE_TAG="latest"
REGISTRY_HOST="192.168.1.222"
REGISTRY_PORT="32768"
FULL_IMAGE_NAME="${REGISTRY_HOST}:${REGISTRY_PORT}/${IMAGE_NAME}:${IMAGE_TAG}"
BUILDER_NAME="multiarch-builder"

# Función para mostrar mensajes informativos
function info {
    echo -e "\e[32m[INFO]\e[0m $1"
}

# Función para mostrar mensajes de error y salir
function error {
    echo -e "\e[31m[ERROR]\e[0m $1"
    exit 1
}

# Verificar si Docker está instalado
if ! command -v docker &> /dev/null; then
    error "Docker no está instalado. Por favor, instálalo y vuelve a intentarlo."
fi

# Iniciar sesión en el registro privado
info "Iniciando sesión en el registro Docker privado..."
docker login "${REGISTRY_HOST}:${REGISTRY_PORT}" || error "Fallo al autenticar en el registro Docker."

# Verificar si buildx está disponible
if ! docker buildx version &> /dev/null; then
    error "Docker Buildx no está disponible. Asegúrate de tener una versión de Docker que lo soporte."
fi

# Eliminar el builder existente si existe
if docker buildx inspect "${BUILDER_NAME}" &> /dev/null; then
    info "Eliminando el builder existente '${BUILDER_NAME}'..."
    docker buildx rm "${BUILDER_NAME}" || error "Fallo al eliminar el builder '${BUILDER_NAME}'."
fi

# Crear un nuevo builder multi-arquitectura
info "Creando un nuevo builder multi-arquitectura llamado '${BUILDER_NAME}'..."
docker buildx create --name "${BUILDER_NAME}" --use --config ./buildkitd.toml || error "Fallo al crear el builder '${BUILDER_NAME}'."

# Inicializar el builder
info "Inicializando el builder '${BUILDER_NAME}'..."
docker buildx inspect --bootstrap || error "Fallo al inicializar el builder '${BUILDER_NAME}'."

# Construir y empujar la imagen para la arquitectura ARM64
info "Construyendo y empujando la imagen para la arquitectura ARM64..."
docker buildx build --platform linux/arm64 \
    -t "${FULL_IMAGE_NAME}" \
    --push \
    . || error "Fallo al construir y empujar la imagen."

info "Imagen '${FULL_IMAGE_NAME}' construida y empujada exitosamente."
