
# Daily Manager Go - Ruleta de Turnos

Este proyecto es una aplicación de escritorio en Go utilizando el framework Fyne, que implementa una ruleta de turnos para una serie de participantes. La aplicación permite seleccionar aleatoriamente un turno, mostrar el turno actual y mantener un registro del tiempo utilizado por cada participante.

## Requisitos

- Go 1.16 o superior
- Bibliotecas de terceros:
  - `fyne.io/fyne/v2`
  - `github.com/hegedustibor/htgo-tts`
  - `image/color`
  - `sync`
  - `math/rand`
  - `os/exec`
- `mplayer` para la reproducción de audio

## Instalación

1. Clona este repositorio:
   ```bash
   git clone https://github.com/facephi/daily-manager-go.git
   cd ruleta-de-turnos
   ```

2. Instala las dependencias:
   ```bash
   go mod tidy
   ```

3. Instala `mplayer` utilizando `brew`:
   ```bash
   brew install mplayer
   ```

4. Ejecuta la aplicación:
   ```bash
   go run main.go
   ```

## Uso

Al ejecutar la aplicación, se abrirá una ventana con la lista de turnos y botones para iniciar y pausar la ruleta.

### Interfaz de Usuario

- **Lista de Turnos:** Muestra todos los turnos con su estado (completado, seleccionado o pendiente).
- **Botón Girar Ruleta:** Selecciona aleatoriamente un turno disponible.
- **Botón Pausar:** Pausa o reanuda el tiempo del turno actual.

### Funcionalidades

- **Selección de Turno:** La ruleta selecciona un turno de manera aleatoria (excluyendo el turno de introducción de Joseca después de la primera vez).
- **Tiempo de Turno:** Registra el tiempo que cada participante utiliza durante su turno.
- **Pausar/Reanudar:** Permite pausar y reanudar el conteo del tiempo de un turno.
- **Audio y Texto a Voz:** Utiliza la biblioteca `htgo-tts` para reproducir audios y anunciar el turno seleccionado.

## Código Principal

El archivo principal `main.go` incluye toda la lógica de la aplicación. Aquí hay una breve descripción de las funciones clave:

- `initializeTurnos()`: Inicializa la lista de turnos con los participantes.
- `initializeUI(w fyne.Window)`: Configura la interfaz de usuario.
- `togglePause()`: Pausa o reanuda el tiempo del turno actual.
- `selectJoseca()`: Selecciona el turno de introducción de Joseca y reproduce un mensaje de audio.
- `selectRandomTurno()`: Selecciona aleatoriamente un turno disponible.
- `animateSelection(finalIndex int, availableTurnos []int)`: Anima la selección de un turno.
- `completeCurrentTurno()`: Marca el turno actual como completado.
- `updateCurrentTurnoTime()`: Actualiza el tiempo del turno actual.
- `sortTurnosByTime()`: Ordena los turnos por tiempo.
- `playAudio(filePath string)`: Reproduce un archivo de audio utilizando `mplayer`.

## Contribuir

Si deseas contribuir a este proyecto, por favor sigue los siguientes pasos:

1. Haz un fork del repositorio.
2. Crea una nueva rama (`git checkout -b feature/nueva-funcionalidad`).
3. Realiza tus cambios y haz commit (`git commit -am 'Añadir nueva funcionalidad'`).
4. Sube los cambios a tu fork (`git push origin feature/nueva-funcionalidad`).
5. Abre un Pull Request.

## Licencia

Este proyecto está licenciado bajo la [MIT License](LICENSE).

## Autor

- [SDK Mobile Team](https://github.com/tu_usuario)

¡Gracias por usar la Ruleta de Turnos!
