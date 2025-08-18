var socket = null;
var twEmoteMap = new Map();
var ytEmoteMap = new Map();

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
    if(command.command === 'setNewUserId' && command.platform == 'twitch') {
        twEmoteMap = new Map();
        fillTwitchEmoteMap(twEmoteMap, command.id);
    }
    if(command.command === 'setNewUserId' && command.platform == 'youtube') {
        ytEmoteMap = new Map();
        fillYoutubeEmoteMap(ytEmoteMap, command.id)
    }
    if(command.command === 'refresh') {
        window.location.reload();
    }
}


function handleNewMessage(message) {
    switch (message.platform) {
        case 'youtube':
            breakMessage(message, ytEmoteMap);
            break;
        case 'twitch':
            breakMessage(message, twEmoteMap);
            break;
        default:
            break;
    }
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
    container.appendChild(createFooterMessageNode(message));
    return container
}

function createHeaderMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-head-container');

    container.appendChild(createHeadeBadgesrMessageNode(message));

    const name = document.createElement('div');
    name.classList.add('message-head-name');
    name.innerText = message.userName;
    container.appendChild(name);

    return container
}

function createHeadeBadgesrMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-head-badges-container');

    message.badges.forEach(badge => {
        const img = document.createElement('img');
        img.src = badge.ImgSrc;
        img.classList.add('message-badge-img');
        img.setAttribute('data-tooltip', badge.Name);
        container.appendChild(img);
    });

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

function createFooterMessageNode(message) {
    const container = document.createElement('div');
    container.classList.add('message-footer-container');
    const img = document.createElement('img');
    img.src = '/platform_icons/yt.png';
    if(message.platform === 'twitch') {
        img.src = '/platform_icons/tw.png';
    }
    img.classList.add('platform-icon-img');
    container.appendChild(img);
    return container
}


window.addEventListener('load', openWebSocket);
