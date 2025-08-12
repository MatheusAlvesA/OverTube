async function fillEmoteMap(map, twitchChanId) {
    await getGlobalEmotes(map);
    await getChannelEmotes(twitchChanId, map);
}

async function getChannelEmotes(twitchChanId, map) {
    const bttv = await fetch('https://api.betterttv.net/3/cached/users/twitch/' + twitchChanId);
    const bttvJson = await bttv.json();
    bttvJson.sharedEmotes.forEach(emote => {
        map.set(emote.code, 'https://cdn.betterttv.net/emote/' + emote.id + '/2x.' + emote.imageType);
    });

    const ffz = await fetch('https://api.betterttv.net/3/cached/frankerfacez/users/twitch/' + twitchChanId);
    const ffzJson = await ffz.json();
    ffzJson.forEach(emote => {
        map.set(emote.code, emote.images['4x']);
    });

    const sevenTv = await fetch('https://7tv.io/v3/users/twitch/' + twitchChanId);
    const sevenTvJson = await sevenTv.json();
    sevenTvJson.emote_set.emotes.forEach(emote => {
        map.set(emote.name, 'https:' + emote.data.host.url + '/' + emote.data.host.files[1].name);
    });
}

async function getGlobalEmotes(map) {
    const globalBttv = await fetch('https://api.betterttv.net/3/cached/emotes/global')
    const globalBttvJson = await globalBttv.json()
    globalBttvJson.forEach(emote => {
        const imageUrl = 'https://cdn.betterttv.net/emote/' + emote.id + '/2x.' + emote.imageType;
        map.set(emote.code, imageUrl);
    })

    const globalFfz = await fetch('https://api.betterttv.net/3/cached/frankerfacez/emotes/global');
    const globalFfzJson = await globalFfz.json();
    globalFfzJson.forEach(emote => {
        map.set(emote.code, emote.images['4x']);
    });

    const global7Z = await fetch('https://7tv.io/v3/emote-sets/global');
    const global7ZJson = await global7Z.json();
    global7ZJson.emotes.forEach(emote => {
        map.set(emote.name, 'https:' + emote.data.host.url + '/' + emote.data.host.files[1].name);
    });
}

function breakMessage(message, map) {
    const parts = [];
    message.messageParts.forEach(part => {
        if(part.PartType !== 'text') {
            parts.push(part);
            return;
        }
        const split = part.Text.split(' ');
        let currentTextAccumulator = '';
        for(const word of split) {
            const emote = getEmote(word, map);
            if(emote) {
                if(currentTextAccumulator !== '') {
                    parts.push({
                        PartType: 'text',
                        Text: currentTextAccumulator.slice(0, -1),
                        EmoteName: '',
                        EmoteImgUrl: '',
                    });
                    currentTextAccumulator = '';
                }
                parts.push({
                    PartType: 'emote',
                    Text: word,
                    EmoteName: word,
                    EmoteImgUrl: emote,
                });
            } else {
                currentTextAccumulator += word + ' ';
            }
        }
        if(currentTextAccumulator !== '') {
            parts.push({
                PartType: 'text',
                Text: currentTextAccumulator.slice(0, -1),
                EmoteName: '',
                EmoteImgUrl: '',
            });
        }
    });
    message.messageParts = parts;
}

function getEmote(word, map) {
    const cleanWord = word.replace(/[^A-Za-z0-9:]/g, '');
    if(map.has(cleanWord)) {
        return map.get(cleanWord);
    }
    return null;
}
