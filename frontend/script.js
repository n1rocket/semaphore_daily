(function () {
  let socket;
  let isMaster = false;
  let userName = "";
  let currentState = {
    meetingStarted: false,
    semaphoreGreen: false,
  };
  let isSpeaking = false;
  let speakingStartTime;
  let hasPressedButton = false;

  const app = document.getElementById("app");

  function renderLogin() {
    app.innerHTML = `
          <h1>Daily Meeting Manager</h1>
          <input type="text" id="nameInput" placeholder="Ingresa tu nombre" />
          <button id="joinButton">Unirse</button>
      `;

    document
      .getElementById("joinButton")
      .addEventListener("click", joinMeeting);
  }

  function joinMeeting() {
    userName = document.getElementById("nameInput").value.trim();

    if (userName === "") {
      alert("Por favor, ingresa tu nombre.");
      return;
    }

    console.log("Usuario ingresando: " + userName);

    connectWebSocket();
  }

  function connectWebSocket() {
    socket = new WebSocket("ws://localhost:8080/ws");

    socket.addEventListener("open", () => {
      console.log("Conexión WebSocket abierta.");

      // Enviar mensaje inicial con el nombre
      const initMessage = {
        name: userName,
      };
      console.log("Enviando mensaje inicial:", initMessage);
      socket.send(JSON.stringify(initMessage));
    });

    socket.addEventListener("message", (event) => {
      const msg = JSON.parse(event.data);
      console.log("Mensaje recibido del servidor:", msg);
      handleMessage(msg);
    });

    socket.addEventListener("close", () => {
      console.log("Conexión WebSocket cerrada.");
      alert("Conexión cerrada. Por favor, recarga la página.");
    });
  }

  function renderMeeting() {
    console.log("Renderizando la interfaz de la reunión. isMaster=" + isMaster);
    app.innerHTML = `
          <h2>Bienvenido, ${userName}${isMaster ? " (Master)" : ""}</h2>
          <div id="roomState"></div>
          <div id="semaphore" class="${
            currentState.semaphoreGreen ? "green" : ""
          }"></div>
          ${
            !isMaster
              ? '<button id="pressButton" disabled>Presiona el botón</button>'
              : ""
          }
          <div id="connectedUsers"></div>
          <div id="turnOrder"></div>
          ${isMaster ? renderMasterControls() : ""}
          <div id="currentSpeaker"></div>
          ${
            !isMaster
              ? '<button id="endTurnButton" class="hidden">Terminar Turno</button>'
              : ""
          }
      `;

    if (!isMaster) {
      document
        .getElementById("pressButton")
        .addEventListener("click", pressButton);
      document
        .getElementById("endTurnButton")
        .addEventListener("click", endTurn);
    }

    if (isMaster) {
      document
        .getElementById("startMeetingButton")
        .addEventListener("click", startMeeting);
      document
        .getElementById("toggleSemaphoreButton")
        .addEventListener("click", toggleSemaphore);
      document
        .getElementById("skipTurnButton")
        .addEventListener("click", skipTurn);
      document
        .getElementById("resetMeetingButton")
        .addEventListener("click", resetMeeting);
    }

    updateRoomState();
    updateSemaphore();
  }

  function renderMasterControls() {
    return `
          <div class="master-controls">
              <button id="startMeetingButton">Iniciar Reunión</button>
              <button id="toggleSemaphoreButton">Alternar Semáforo</button>
              <button id="skipTurnButton">Saltar Turno</button>
              <button id="resetMeetingButton">Reiniciar Reunión</button>
          </div>
      `;
  }

  function handleMessage(msg) {
    console.log("handleMessage:", msg);

    switch (msg.type) {
      case "initial_role":
        isMaster = msg.payload.isMaster;
        console.log("Rol recibido: isMaster=" + isMaster);
        renderMeeting();
        break;
      case "you_are_master":
        isMaster = true;
        console.log("Ahora eres el master.");
        renderMeeting();
        break;
      case "user_list":
        console.log("Lista de usuarios conectados actualizada.");
        updateUserList(msg.payload);
        break;
      case "semaphore_toggled":
        // Obsoleto, ahora usamos 'meeting_state'
        break;
      case "meeting_state":
        currentState.meetingStarted = msg.payload.meetingStarted;
        currentState.semaphoreGreen = msg.payload.semaphoreGreen;
        console.log("Estado de la reunión actualizado:", currentState);
        updateRoomState();
        updateSemaphore();
        break;
      case "turn_order":
        console.log("Orden de turnos actualizado.");
        updateTurnOrder(msg.payload);
        break;
      case "next_turn":
        console.log("Es el turno de: " + msg.payload);
        startNextTurn(msg.payload);
        break;
      case "turn_started":
        console.log("Turno iniciado para: " + msg.payload);
        break;
      case "meeting_started":
        console.log("La reunión ha comenzado.");
        currentState.meetingStarted = true;
        updateRoomState();
        break;
      case "meeting_reset":
        console.log("La reunión ha sido reiniciada.");
        alert("La reunión ha sido reiniciada.");
        location.reload();
        break;
      case "tts":
        console.log("Anunciando siguiente orador: " + msg.payload);
        announceNextSpeaker(msg.payload);
        break;
      case "meeting_finished":
        console.log("La reunión ha finalizado.");
        showUserTimes(msg.payload);
        break;
      default:
        console.log("Tipo de mensaje desconocido:", msg.type);
    }
  }

  function updateRoomState() {
    const roomStateElement = document.getElementById("roomState");
    if (currentState.meetingStarted) {
      roomStateElement.innerHTML = "<p>Estado: Reunión en curso</p>";
    } else {
      roomStateElement.innerHTML =
        "<p>Estado: Esperando a que el master inicie la reunión</p>";
    }
  }

  function updateSemaphore() {
    const semaphore = document.getElementById("semaphore");
    const pressButton = document.getElementById("pressButton");

    if (currentState.semaphoreGreen) {
      semaphore.classList.add("green");
      if (!isMaster && !currentState.meetingStarted && !hasPressedButton) {
        pressButton.disabled = false;
      }
    } else {
      semaphore.classList.remove("green");
      if (!isMaster) {
        pressButton.disabled = true;
      }
    }
  }

  function pressButton() {
    console.log("Presionaste el botón para unirte al orden de turnos.");
    socket.send(
      JSON.stringify({
        type: "press_button",
      })
    );
    hasPressedButton = true;
    document.getElementById("pressButton").disabled = true;
  }

  function updateUserList(users) {
    const connectedUsersElement = document.getElementById("connectedUsers");
    connectedUsersElement.innerHTML = "<h3>Usuarios Conectados:</h3>";
    users.forEach((name) => {
      const p = document.createElement("p");
      p.textContent = name;
      connectedUsersElement.appendChild(p);
    });
  }

  function updateTurnOrder(order) {
    const turnOrderElement = document.getElementById("turnOrder");
    turnOrderElement.innerHTML =
      '<h3>Orden de Turnos:</h3><ul id="turnOrderList"></ul>';

    const turnOrderList = document.getElementById("turnOrderList");
    order.forEach((name) => {
      const li = document.createElement("li");
      li.textContent = name;
      turnOrderList.appendChild(li);
    });

    if (isMaster) {
      // Hacer la lista ordenable
      new Sortable(turnOrderList, {
        animation: 150,
        onEnd: () => {
          const newOrder = Array.from(turnOrderList.children).map(
            (li) => li.textContent
          );
          console.log("Nuevo orden de turnos:", newOrder);
          socket.send(
            JSON.stringify({
              type: "reorder_turn_order",
              payload: newOrder,
            })
          );
        },
      });
    }
  }

  function startNextTurn(name) {
    const currentSpeakerElement = document.getElementById("currentSpeaker");
    currentSpeakerElement.innerHTML = `<h3>Está hablando: ${name}</h3>`;

    if (name === userName) {
      // Es nuestro turno
      isSpeaking = true;
      speakingStartTime = new Date();
      document.getElementById("endTurnButton").classList.remove("hidden");
      // Iniciar el turno
      console.log("Es tu turno de hablar.");
      socket.send(
        JSON.stringify({
          type: "start_turn",
        })
      );
    } else {
      isSpeaking = false;
      document.getElementById("endTurnButton").classList.add("hidden");
    }
  }

  function endTurn() {
    if (isSpeaking) {
      const turnTime = new Date() - speakingStartTime;
      console.log(
        "Terminaste tu turno. Tiempo utilizado: " +
          turnTime / 1000 +
          " segundos."
      );
      socket.send(
        JSON.stringify({
          type: "end_turn",
          payload: turnTime,
        })
      );
      isSpeaking = false;
      document.getElementById("endTurnButton").classList.add("hidden");
    }
  }

  function announceNextSpeaker(name) {
    const utterance = new SpeechSynthesisUtterance(`Es el turno de ${name}`);
    speechSynthesis.speak(utterance);
  }

  function showUserTimes(times) {
    app.innerHTML = "<h2>Reunión Finalizada</h2>";
    const timesDiv = document.createElement("div");
    timesDiv.innerHTML = "<h3>Tiempos Utilizados:</h3>";
    const ul = document.createElement("ul");
    times.forEach((user) => {
      const li = document.createElement("li");
      li.textContent = `${user.name}: ${user.turnTime.toFixed(2)} segundos`;
      ul.appendChild(li);
    });
    timesDiv.appendChild(ul);
    app.appendChild(timesDiv);
  }

  // Funciones para el master
  function startMeeting() {
    console.log("Iniciaste la reunión.");
    socket.send(
      JSON.stringify({
        type: "start_meeting",
      })
    );
  }

  function toggleSemaphore() {
    console.log("Alternaste el semáforo.");
    socket.send(
      JSON.stringify({
        type: "toggle_semaphore",
      })
    );
  }

  function skipTurn() {
    console.log("Saltaste el turno actual.");
    socket.send(
      JSON.stringify({
        type: "skip_turn",
      })
    );
  }

  function resetMeeting() {
    console.log("Reiniciaste la reunión.");
    socket.send(
      JSON.stringify({
        type: "reset_meeting",
      })
    );
  }

  // Inicializar la aplicación
  renderLogin();
})();
