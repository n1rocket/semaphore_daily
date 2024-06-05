package main

import (
	"fmt"
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/hegedustibor/htgo-tts/handlers"
	"github.com/hegedustibor/htgo-tts/voices"
	"image/color"
	"math/rand"
	"os/exec"
	"sort"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Turno struct {
	Palabra      string
	Completado   bool
	Tiempo       time.Duration
	Seleccionado bool
}

var (
	turnos             []Turno
	selectedLabel      *canvas.Text
	pendingList        *widget.List
	startTime          time.Time
	elapsedPausedTime  time.Duration
	currentTurnoIndex  int = -1
	ticker             *time.Ticker
	animating          bool
	mutex              sync.Mutex
	introduccionJoseca = true
	paused             bool
	pausedButton       *widget.Button
)

func main() {
	a := app.New()
	w := a.NewWindow("Ruleta de Turnos")

	initializeTurnos()
	initializeUI(w)

	w.Resize(fyne.NewSize(400, 600))
	w.ShowAndRun()
}

func initializeTurnos() {
	turnos = []Turno{
		{"Introducción de Joseca", false, 0, false},
		{"Jose Antonio", false, 0, false},
		{"Dani", false, 0, false},
		{"Leti", false, 0, false},
		{"Alex", false, 0, false},
		{"Raúl", false, 0, false},
		{"Javi", false, 0, false},
		{"Rubén", false, 0, false},
		{"Jorge", false, 0, false},
		{"Armando", false, 0, false},
		{"Carlos", false, 0, false},
		{"Leandro", false, 0, false},
	}
}

func initializeUI(w fyne.Window) {
	pendingList = widget.NewList(
		func() int { return len(turnos) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			updateListLabel(i, o.(*widget.Label))
		},
	)

	selectedLabel = canvas.NewText("", color.White)
	selectedLabel.Alignment = fyne.TextAlignCenter
	selectedLabel.TextStyle = fyne.TextStyle{Bold: true}
	selectedLabel.TextSize = 24

	startButton := widget.NewButton("Girar Ruleta", func() {
		mutex.Lock()
		if !animating {
			animating = true
			mutex.Unlock()
			if introduccionJoseca {
				selectJoseca()
			} else {
				selectRandomTurno()
			}
		} else {
			mutex.Unlock()
		}
	})

	pausedButton = widget.NewButton("Pausar", togglePause)

	content := container.NewVBox(
		selectedLabel,
		layout.NewSpacer(),
		startButton,
		pausedButton,
	)

	mainContent := container.NewBorder(
		nil,
		content,
		nil,
		nil,
		pendingList,
	)

	w.SetContent(mainContent)
}

func togglePause() {
	mutex.Lock()
	defer mutex.Unlock()

	if paused {
		// Reanudar
		startTime = time.Now().Add(-elapsedPausedTime)
		ticker = time.NewTicker(time.Second)
		go func() {
			for range ticker.C {
				updateCurrentTurnoTime()
			}
		}()
		pausedButton.SetText("Pausar")
		paused = false
	} else {
		// Pausar
		if ticker != nil {
			ticker.Stop()
		}
		elapsedPausedTime = time.Since(startTime)
		pausedButton.SetText("Reanudar")
		paused = true
	}
}

func updateListLabel(i int, label *widget.Label) {
	if turnos[i].Completado {
		label.SetText(fmt.Sprintf("✔ %s - %v", turnos[i].Palabra, turnos[i].Tiempo))
		label.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		if turnos[i].Seleccionado {
			label.SetText(fmt.Sprintf("%s - %v", turnos[i].Palabra, turnos[i].Tiempo))
			label.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
		} else {
			label.SetText(turnos[i].Palabra)
			label.TextStyle = fyne.TextStyle{}
		}
	}
	label.Refresh()
}

func selectJoseca() {
	// Seleccionar Introducción de Joseca
	turnos[0].Seleccionado = true
	startTime = time.Now()
	elapsedPausedTime = 0
	currentTurnoIndex = 0
	pendingList.Refresh()

	if ticker != nil {
		ticker.Stop()
	}
	ticker = time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			updateCurrentTurnoTime()
		}
	}()

	mutex.Lock()
	animating = false
	introduccionJoseca = false // Una vez completado, no será principal de nuevo
	mutex.Unlock()

	speech := htgotts.Speech{Folder: "audio", Language: voices.Spanish, Handler: &handlers.MPlayer{}}
	err := speech.Speak("¡Joseca!")
	if err != nil {
		return
	}

	/*
		err = playAudio("audio/feria.mp3")
		if err != nil {
			log.Fatalf("Error al reproducir el audio: %v", err)
		}
	*/

}

func selectRandomTurno() {
	if currentTurnoIndex >= 0 {
		completeCurrentTurno()
	}

	availableTurnos := getAvailableTurnos()

	if len(availableTurnos) == 0 {
		selectedLabel.Text = "Todos los turnos completados"
		selectedLabel.Refresh()
		animating = false

		speech := htgotts.Speech{Folder: "audio", Language: voices.Spanish, Handler: &handlers.MPlayer{}}
		err := speech.Speak("¡TODO LISTO!")
		if err != nil {
			return
		}
		return
	}

	finalIndex := availableTurnos[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(availableTurnos))]
	go animateSelection(finalIndex, availableTurnos)
}

func getAvailableTurnos() []int {
	var availableTurnos []int
	for i, turno := range turnos {
		if !turno.Completado && i != 0 { // Excluir el primer turno de Joseca
			availableTurnos = append(availableTurnos, i)
		}
	}
	return availableTurnos
}

func animateSelection(finalIndex int, availableTurnos []int) {
	animationDuration := 2 * time.Second
	stepDuration := animationDuration / time.Duration(len(availableTurnos)*2)

	for i := 0; i < len(availableTurnos)*2; i++ {
		time.Sleep(stepDuration)
		updateSelection(i, availableTurnos)
	}
	finalizeSelection(finalIndex)
}

func updateSelection(i int, availableTurnos []int) {
	currentIndex := availableTurnos[i%len(availableTurnos)]
	for j := range turnos {
		turnos[j].Seleccionado = j == currentIndex
	}
	pendingList.Refresh()
}

func finalizeSelection(finalIndex int) {
	for j := range turnos {
		turnos[j].Seleccionado = false
	}
	turnos[finalIndex].Seleccionado = true
	startTime = time.Now()
	elapsedPausedTime = 0
	currentTurnoIndex = finalIndex
	pendingList.Refresh()

	if ticker != nil {
		ticker.Stop()
	}
	ticker = time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			updateCurrentTurnoTime()
		}
	}()

	mutex.Lock()
	animating = false
	mutex.Unlock()

	speech := htgotts.Speech{Folder: "audio", Language: voices.Spanish, Handler: &handlers.MPlayer{}}
	err := speech.Speak(turnos[finalIndex].Palabra)
	if err != nil {
		return
	}
}

func completeCurrentTurno() {
	if currentTurnoIndex >= 0 {
		turnos[currentTurnoIndex].Completado = true
		turnos[currentTurnoIndex].Tiempo = time.Since(startTime).Truncate(time.Second)
		turnos[currentTurnoIndex].Seleccionado = false
		currentTurnoIndex = -1
		if ticker != nil {
			ticker.Stop()
		}
	}
	sortTurnosByTime()
	pendingList.Refresh()
}

func updateCurrentTurnoTime() {
	if currentTurnoIndex >= 0 {
		turnos[currentTurnoIndex].Tiempo = time.Since(startTime).Truncate(time.Second)
		pendingList.Refresh()
	}
}

func sortTurnosByTime() {
	sort.Slice(turnos, func(i, j int) bool {
		return turnos[i].Tiempo > turnos[j].Tiempo
	})
}

func playAudio(filePath string) error {
	cmd := exec.Command("mplayer", filePath)
	err := cmd.Run()
	return err
}
