# Usa una imagen base con Go preinstalado
FROM golang:1.22

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos necesarios
COPY . .

# Descarga las dependencias
RUN go mod download

# Compila la aplicación
RUN go build -o app main.go

# Expone el puerto utilizado por la aplicación
EXPOSE 8080

# Comando para ejecutar la aplicación
CMD ["./app"]