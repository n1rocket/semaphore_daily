# Daily Meeting Manager - Gestor de Reuniones Diarias

Este proyecto es una aplicación web diseñada para facilitar la gestión de reuniones diarias. Permite coordinar el turno de participación de los usuarios, manejar el flujo de la reunión y proporcionar herramientas interactivas para mejorar la colaboración en equipo.

## Descripción

La aplicación está construida utilizando **Go** en el backend con el framework **Gin** y en el frontend emplea **HTML, CSS y JavaScript**. Para la comunicación en tiempo real entre el servidor y los clientes se utiliza **WebSockets**, lo que permite actualizar el estado de la reunión y las interacciones de los usuarios al instante.

## Características Principales

### Gestión de Usuarios

- Los participantes pueden unirse a la reunión ingresando su nombre.
- El primer usuario en conectarse es asignado como **Master**, teniendo controles adicionales para gestionar la reunión.

### Control de Turnos

- Implementa una mecánica para determinar el orden de intervención de los participantes mediante un **semáforo virtual** y un **botón de participación**.
- Los usuarios pueden presionar el botón cuando el semáforo está en verde para unirse a la lista de turnos.

### Semáforo Virtual

- El semáforo cambia de rojo a verde después de un tiempo aleatorio, indicando a los participantes cuándo pueden presionar el botón.
- Añade una dinámica interactiva y aleatoria para iniciar las intervenciones.

### Interfaz Intuitiva

- Diseño claro y sencillo que muestra la lista de usuarios conectados, el orden de turnos y el orador actual.
- Botones y controles fáciles de usar para interactuar con la aplicación.

### Funciones del Master

- Iniciar y reiniciar la reunión.
- Iniciar el semáforo.
- Saltar turnos.
- Añadir usuarios virtuales para pruebas o simulaciones.

## Requisitos

### Backend

- **Go 1.16** o superior.
- Dependencias:
  - `github.com/gin-gonic/gin`
  - `github.com/gorilla/websocket`

### Frontend

- Navegador web moderno y actualizado.
- Conexión a Internet para cargar librerías externas como **SortableJS**.

## Instalación

1. **Clonar el repositorio:**

   ```bash
   git clone https://github.com/usuario/daily-meeting-manager.git
   cd daily-meeting-manager
   ```

2. **Instalar las dependencias de Go:**

   ```bash
   go mod tidy
   ```

3. **Ejecutar la aplicación:**

   ```bash
   go run main.go
   ```

4. La aplicación estará disponible en [http://localhost:8080](http://localhost:8080).

## Uso

### Acceder a la Aplicación

- Abre un navegador web y navega a [http://localhost:8080](http://localhost:8080).

### Unirse a la Reunión

1. Ingresa tu nombre en el campo proporcionado y haz clic en **"Unirse"**.
2. Si eres el primer usuario en unirte, serás el **Master** de la reunión.

### Interfaz de Usuario

- **Semáforo:**
  - Indica cuándo los participantes pueden presionar el botón para unirse al turno.
  - Cambia de rojo a verde después de un tiempo aleatorio.
- **Botón "Presiona el botón":**
  - Disponible cuando el semáforo está en verde.
  - Al presionarlo, te unes a la lista de turnos.
- **Lista de Usuarios Conectados:**
  - Muestra todos los participantes actualmente en la reunión.
- **Orden de Turnos:**
  - Muestra el orden en el que los participantes hablarán.
- **Orador Actual:**
  - Indica quién está hablando en este momento.

### Funciones del Master

- **Iniciar Reunión:** Comienza la reunión y permite que otros usuarios se unan.
- **Iniciar Semáforo:** Activa el semáforo para que los participantes puedan presionar el botón.
- **Saltar Turno:** Omite al orador actual y pasa al siguiente en la lista.
- **Añadir Usuario Virtual:** Agrega un participante virtual para pruebas o demostraciones.
- **Reiniciar Reunión:** Restablece el estado de la reunión y desconecta a todos los usuarios.

## Arquitectura del Proyecto

### Backend (`main.go`)

- **Manejo de Usuarios:**
  - Conecta y desconecta usuarios utilizando WebSockets.
  - Asigna roles y mantiene el estado de cada participante.
- **Estado de la Reunión:**
  - Controla el inicio y reinicio de la reunión.
  - Gestiona el semáforo y el turno de los participantes.
- **Comunicación en Tiempo Real:**
  - Envía y recibe mensajes JSON a través de WebSockets para actualizar el estado de la aplicación en los clientes.

### Frontend (`index.html`, `styles.css`, `script.js`)

- **Interfaz de Usuario:**
  - HTML estructurado para mostrar los componentes de la aplicación.
  - CSS para estilizar y mejorar la experiencia visual.
- **Interactividad:**
  - JavaScript para manejar eventos, actualizar la interfaz y comunicarse con el servidor.
  - Utilización de **SortableJS** para permitir la reorganización interactiva de elementos si es necesario.
- **Comunicación con el Servidor:**
  - Establece una conexión WebSocket para enviar acciones del usuario y recibir actualizaciones.

## Exponer con NGROK al exterior

```bash
ngrok http http://localhost:8080
```

## Contribuir

Si deseas contribuir al proyecto:

1. Realiza un fork del repositorio.
2. Crea una nueva rama para tu funcionalidad:

   ```bash
   git checkout -b feature/nueva-funcionalidad
   ```

3. Realiza tus cambios y realiza commits descriptivos.
4. Sube tus cambios al repositorio remoto:

   ```bash
   git push origin feature/nueva-funcionalidad
   ```

5. Abre un **Pull Request** explicando tus modificaciones.

## Licencia

Este proyecto está bajo la Licencia MIT. Consulta el archivo `LICENSE` para más detalles.

## Autor

**Equipo de Desarrollo**

---

¡Gracias por utilizar **Daily Meeting Manager**!
