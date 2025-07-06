var socket = null;
function openWebSocket() {
    socket = new WebSocket("ws://localhost:1336/ws");
    socket.onopen = (event) => console.log("Websocket connected!");
    socket.onmessage = (event) => handleNewPayload(event.data);

    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
    };
    socket.onclose = (event) => {
        console.log("WebSocket connection closed:", event);
    };
}

function handleNewPayload(payload) {
    const parsed = JSON.parse(payload);
    if(parsed.type === "msg") {
        handleNewMessage(parsed);
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
    container.appendChild(createHeaderMessageNode(message));
    container.appendChild(createBodyMessageNode(message));
    return container
}

function createHeaderMessageNode(message) {
    const container = document.createElement('div');
    container.innerText = message.userName;
    return container
}

function createBodyMessageNode(message) {
    const container = document.createElement('div');
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
