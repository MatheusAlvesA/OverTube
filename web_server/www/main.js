var socket = null;
function openWebSocket() {
    if(socket != null) return;

    socket = new WebSocket("ws://localhost:1336/ws");
    socket.onopen = (event) => {
        console.log("Websocket connected!");
        document.getElementById('alert-disconnected').style.display = 'none';
    }
    socket.onmessage = (event) => handleNewPayload(event.data);

    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
        document.getElementById('alert-disconnected').style.display = 'flex';
        socket = null;
        setTimeout(() => openWebSocket(), 1000);
    };
    socket.onclose = (event) => {
        console.log("WebSocket connection closed:", event);
        document.getElementById('alert-disconnected').style.display = 'flex';
        socket = null;
        setTimeout(() => openWebSocket(), 1000);
    };
}

function handleNewPayload(payload) {
    const parsed = JSON.parse(payload);
    if(parsed.type === "msg") {
        handleNewMessage(parsed);
    }
    if(parsed.type === "cmd") {
        handleNewCommand(parsed);
    }
}

function handleNewCommand(command) {
    if(command.command === 'ping') {
        socket.send(JSON.stringify({'command': 'pong'}));
    }
}


function handleNewMessage(message) {
    const node = createMessageNode(message);
    document.getElementById('messagesContainer').appendChild(node);
    deleteOldMessages();
    window.scrollTo(0, document.body.scrollHeight);
}

function deleteOldMessages() {
    const container = document.getElementById('messagesContainer');
    const nToRemove = container.children.length - 100;
    for(let i = 0; i < nToRemove; i++) {
        container.removeChild(container.firstElementChild);
    }
}

function createMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-container');
    container.appendChild(createHeaderMessageNode(message));
    container.appendChild(createBodyMessageNode(message));
    return container
}

function createHeaderMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-head-container');
    container.innerText = message.userName;
    return container
}

function createBodyMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-body-container');
    message.messageParts.forEach(part => {
        if(part.PartType === "text") {
            const span = document.createElement('span');
            span.innerHTML = part.Text;
            container.appendChild(span);
        } else {
            const img = document.createElement('img');
            img.src = part.EmoteImgUrl;
            img.classList.add('message-emote-img');
            container.appendChild(img);
        }
    });
    return container
}


window.addEventListener('load', openWebSocket);
