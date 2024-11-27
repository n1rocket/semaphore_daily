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
    const wsProtocol =
      window.location.protocol === "https:" ? "wss://" : "ws://";
    socket = new WebSocket(`${wsProtocol}${window.location.host}/ws`);

    socket.addEventListener("open", () => {
      // Enviar el nombre de usuario al servidor
      const initMsg = JSON.stringify({
        name: userName,
      });
      console.log("Enviando mensaje inicial: " + initMsg);
      socket.send(initMsg);
    });

    socket.addEventListener("message", (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (error) {
        console.error("Error al parsear el mensaje:", event.data, error);
      }
    });

    socket.addEventListener("close", () => {
      console.log("Conexión cerrada.");
      alert("Conexión con el servidor perdida.");
      location.reload();
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
          <button id="pressButton" disabled>Presiona el botón</button>
          <div id="connectedUsers"></div>
          <div id="turnOrder"></div>
          ${isMaster ? renderMasterControls() : ""}
          <div id="currentSpeaker"></div>
          <button id="endTurnButton" class="hidden">Terminar Turno</button>
      `;

    document
      .getElementById("pressButton")
      .addEventListener("click", pressButton);

    document.getElementById("endTurnButton").addEventListener("click", endTurn);

    if (isMaster) {
      document
        .getElementById("startMeetingButton")
        .addEventListener("click", startMeeting);

      document
        .getElementById("startSemaphoreButton")
        .addEventListener("click", startSemaphore);

      document
        .getElementById("skipTurnButton")
        .addEventListener("click", skipTurn);

      document
        .getElementById("resetMeetingButton")
        .addEventListener("click", resetMeeting);

      document
        .getElementById("addVirtualUserButton")
        .addEventListener("click", addVirtualUser);
    }

    updateRoomState();
    updateSemaphore();
  }

  function renderMasterControls() {
    return `
          <div class="master-controls">
              <button id="startMeetingButton">Iniciar Reunión</button>
              <button id="startSemaphoreButton">Iniciar Semáforo</button>
              <button id="skipTurnButton">Saltar Turno</button>
              <button id="addVirtualUserButton">Añadir Usuario Virtual</button>
              <button id="resetMeetingButton">Reiniciar Reunión</button>
          </div>
      `;
  }

  function handleMessage(msg) {
    console.log("handleMessage:", msg);

    switch (msg.type) {
      case "initial_role":
        console.log("Mensaje initial_role recibido:", msg);
        isMaster = msg.payload.isMaster;
        renderMeeting();
        break;

      case "meeting_state":
        console.log("Mensaje meeting_state recibido:", msg);
        currentState.meetingStarted = msg.payload.meetingStarted;
        currentState.semaphoreGreen = msg.payload.semaphoreGreen;
        updateRoomState();
        updateSemaphore();
        break;

      case "user_list":
        console.log("Mensaje user_list recibido:", msg);
        updateUserList(msg.payload);
        break;

      case "turn_order":
        console.log("Mensaje turn_order recibido:", msg);
        updateTurnOrder(msg.payload);
        break;

      case "next_speaker":
        console.log("Mensaje next_speaker recibido:", msg);
        startNextTurn(msg.payload);
        break;

      case "meeting_end":
        console.log("Mensaje meeting_end recibido:", msg);
        alert(msg.payload); // Mostrar alerta al usuario
        resetInterface(); // Función para reiniciar o limpiar la interfaz
        break;

      case "meeting_reset":
        alert("La reunión ha sido reiniciada por el master.");
        location.reload();
        break;

      default:
        console.log("Tipo de mensaje desconocido:", msg.type);
    }
  }

  function resetInterface() {
    app.innerHTML = `
          <h1>Reunión Finalizada</h1>
          <p>La reunión ha finalizado. Puedes unirte a otra reunión si lo deseas en 3 segundos.</p>
      `;

    // Delay de 3 segundos hacer reload
    setTimeout(() => {
      location.reload();
    }, 3000);
  }

  function updateRoomState() {
    const stateDiv = document.getElementById("roomState");
    stateDiv.textContent = currentState.meetingStarted
      ? "La reunión ha comenzado."
      : "Esperando a que el master inicie la reunión.";
  }

  function updateSemaphore() {
    const semaphoreDiv = document.getElementById("semaphore");
    if (currentState.semaphoreGreen) {
      semaphoreDiv.classList.add("green");
      if (!hasPressedButton) {
        document.getElementById("pressButton").disabled = false;
      }
    } else {
      semaphoreDiv.classList.remove("green");
      document.getElementById("pressButton").disabled = true;
    }
  }

  function pressButton() {
    if (currentState.semaphoreGreen && !hasPressedButton) {
      console.log("Has presionado el botón para tomar el turno.");
      socket.send(
        JSON.stringify({
          type: "press_button",
        })
      );
      hasPressedButton = true;
      document.getElementById("pressButton").disabled = true;
    } else {
      alert("No puedes presionar el botón en este momento.");
    }
  }

  function updateUserList(users) {
    const usersDiv = document.getElementById("connectedUsers");
    usersDiv.innerHTML = "<h3>Usuarios Conectados:</h3><ul>";
    users.forEach((user) => {
      usersDiv.innerHTML += `<li>${user}</li>`;
    });
    usersDiv.innerHTML += "</ul>";
  }

  function updateTurnOrder(order) {
    const turnDiv = document.getElementById("turnOrder");
    turnDiv.innerHTML = "<h3>Orden de Turnos:</h3><ul>";
    order.forEach((user) => {
      turnDiv.innerHTML += `<li>${user}</li>`;
    });
    turnDiv.innerHTML += "</ul>";
  }

  function startNextTurn(name) {
    const speakerDiv = document.getElementById("currentSpeaker");
    speakerDiv.innerHTML = `<h3>Es el turno de: ${name}</h3>`;
    if (name === userName) {
      isSpeaking = true;
      speakingStartTime = Date.now();
      if (!isMaster) {
        const endButton = document.getElementById("endTurnButton");
        endButton.classList.remove("hidden");
      }
    } else {
      isSpeaking = false;
      if (!isMaster) {
        const endButton = document.getElementById("endTurnButton");
        endButton.classList.add("hidden");
      }
    }
  }

  function endTurn() {
    if (isSpeaking) {
      const totalTime = (Date.now() - speakingStartTime) / 1000;
      console.log(`Has terminado tu turno. Tiempo: ${totalTime.toFixed(2)} s`);
      socket.send(
        JSON.stringify({
          type: "end_turn",
        })
      );
      isSpeaking = false;
      const endButton = document.getElementById("endTurnButton");
      endButton.classList.add("hidden");
    }
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

  function startSemaphore() {
    console.log("Iniciaste el semáforo.");
    socket.send(
      JSON.stringify({
        type: "start_semaphore",
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
    if (confirm("¿Estás seguro de reiniciar la reunión?")) {
      console.log("Reiniciaste la reunión.");
      socket.send(
        JSON.stringify({
          type: "reset_meeting",
        })
      );
    }
  }

  function addVirtualUser() {
    console.log("Añadiste un usuario virtual.");
    socket.send(
      JSON.stringify({
        type: "add_virtual_user",
      })
    );
  }

  // Inicializar la aplicación
  renderLogin();
})();
