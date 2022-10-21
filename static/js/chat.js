/// <reference path='./both.js' />
/// <reference path='./consts.js' />

let maxMessageCount = 0;
let inChat = false;
let users = [];
let emotes = {};

// Suggestions
const SuggestionType = {
    None: 0,
    Name: 1,
    Emote: 2
};

Object.freeze(SuggestionType);
let currentSuggestionType = SuggestionType.None;
let currentSuggestion = '';
let filteredSuggestion = [];

function debug() {
    let color = getCookie('color');
    let timestamp = getCookie('timestamp');

    Object.entries({
        maxMessageCount,
        inChat,
        users,
        emotes,
        color,
        timestamp,
    }).forEach(([k, v]) => console.log(k, v));
}

function randomColor() {
    let color = '#';
    for (let i = 0; i < 6; i++) {
        const random = Math.random();
        const bit = (random * 16) | 0;
        color += (bit).toString(16);
    };
    return color;
}

/**
 * @param {string} color
 */
function isValidColor(color) {
    color = color.replace(/^#/, '', color).toLowerCase();
    if (Colors.includes(color)) {
        return true;
    }

    if (ColorRegex.test(color)) {
        hex = color.match(/.{1,2}/g);
        r = parseInt(hex[0], 16);
        g = parseInt(hex[1], 16);
        b = parseInt(hex[2], 16);
        total = r + g + b;
        return total > 0.7 && b / total < 0.7;
    }

    return false;
}

/**
 * @param {string} title
 * @param {string} link
 */
function setPlaying(title, link) {
    if (title !== '') {
        $('#playing').text(title);
        document.title = `${pageTitle} | ${title}`;
    } else {
        $('#playing').text('');
        document.title = pageTitle;
    }

    $('#playing').removeAttr('href');
    if (link !== '') {
        $('#playing').attr('href', link);
    }
}

function getWsUri() {
    port = window.location.port;
    if (port != '') {
        port = `:${port}`;
    }
    proto = location.protocol == 'https:' ? 'wss://' : 'ws://';
    return `${proto}${window.location.hostname}${port}/ws`;
}

/**
 * @param {string} msg
 */
function appendMessages(msg) {
    let msgs = $('#messages').find('div');

    // let's just say that if the max count is less than 1, then the count is infinite
    // the server side should take care of chaking max count ranges
    if (msgs.length > maxMessageCount) {
        msgs.first().remove();
    }

    $('#messages').append(`<div>${msg}</div>`);
    $('#messages').children().last()[0].scrollIntoView({ block: 'end' });
}

function purgeChat() {
    $('#messages').empty()
}

function openChat() {
    console.log('chat opening');
    $('#joinbox').css('display', 'none');
    $('#chat').css('display', 'grid');
    $('#hidden').css('display', '')
    $('#msg').val('');
    $('#msg').focus();
    inChat = true;
}

function closeChat() {
    console.log('chat closing');
    $('#joinbox').css('display', '');
    $('#chat').css('display', 'none');
    $('#hidden').css('display', 'none')
    setNotifyBox('That name was already used!');
    inChat = false;
}

function handleHiddenMessage(data) {
    switch (data.Type) {
        case ClientDataType.CdUsers:
            users = data.Data;
            break;
        case ClientDataType.CdColor:
            setCookie('color', data.Data);
            break;
        case ClientDataType.CdEmote:
            emotes = data.Data;
            break;
        case ClientDataType.CdJoin:
            setNotifyBox('');
            openChat();
            break;
        case ClientDataType.CdNotify:
            setNotifyBox(data.Data);
            break;
        default:
            console.warn('unhandled hidden type', data);
            break;
    }
}

/**
 * @param {*} data
 * @param {bool} isEvent
 */
function handleChatMessage(data, isEvent) {
    msg = data.Message;

    if (isEvent) {
        function nameChangeMsg(forced) {
            let users = data.User.split(':');

            if (users.length < 2) {
                return `<span class="event">Somebody changed their name, but IDK who.</span>`;
            } else {
                if (forced) {
                    return `<span class="event"><span class="name" style="color:${data.Color}">${users[0]}</span> has had their name changed to <span class="name" style="color:${data.Color}">${users[1]}</span> by an admin.</span>`;
                } else {
                    return `<span class="event"><span class="name" style="color:${data.Color}">${users[0]}</span> has changed their name to <span class="name" style="color:${data.Color}">${users[1]}</span>.</span>`;
                }
            }
        }

        switch (data.Event) {
            case EventType.EvKick:
                msg = `<span class="event"><span class="name" style="color:${data.Color}">${data.User}</span> has been kicked.</span>`;
                break;
            case EventType.EvLeave:
                msg = `<span class="event"><span class="name" style="color:${data.Color}">${data.User}</span> has left the chat.</span>`;
                break;
            case EventType.EvBan:
                msg = `<span class="event"><span class="name" style="color:${data.Color}">${data.User}</span> has been banned.</span>`;
                break;
            case EventType.EvJoin:
                msg = `<span class="event"><span class="name" style="color:${data.Color}">${data.User}</span> has joined the chat.</span>`;
                break;
            case EventType.EvNameChange:
                msg = nameChangeMsg(false);
                break;
            case EventType.EvNameChangeForced:
                msg = nameChangeMsg(true);
                break;
        }

    } else {
        function spanMsg(className, content) {
            return `<span class="${className}">${content}</span>`;
        }
        switch (data.Type) {
            case MessageType.MsgAction:
                msg = `<span style="color:${data.Color}">${spanMsg('name', data.From)} ${spanMsg('cmdme', msg)}</span>`;
                break;
            case MessageType.MsgServer:
                msg = spanMsg('announcement', msg);
                break;
            case MessageType.MsgError:
                msg = spanMsg('error', msg);
                break;
            case MessageType.MsgNotice:
                msg = spanMsg('notice', msg);
                break;
            case MessageType.MsgCommandResponse:
                msg = spanMsg('command', msg);
                break;
            case MessageType.MsgCommandError:
                msg = spanMsg('commanderror', msg);
                break;
            default:
                msg = spanMsg('msg', msg);
                switch (data.Level) {
                    case CommandLevel.CmdlMod:
                        msg = `<span><img src="/static/img/mod.png" class="badge" /><span class="name" style="color:${data.Color}">${data.From}</span><b>:</b> ${msg}</span>`;
                        break;
                    case CommandLevel.CmdlAdmin:
                        msg = `<span><img src="/static/img/admin.png" class="badge" /><span class="name" style="color:${data.Color}">${data.From}</span><b>:</b> ${msg}</span>`;
                        break;
                    default:
                        msg = `<span><span class="name" style="color:${data.Color}">${data.From}</span><b>:</b> ${msg}</span>`;
                        break;
                }
                break;
        }
    }

    if (getCookie('timestamp') === 'true' && (data.Type == MessageType.MsgChat || data.Type == MessageType.MsgAction)) {
        let now = new Date();
        let pad = (n) => String(n.toFixed(0)).padStart(2, '0');
        msg = `<span class="time">${pad(now.getHours())}:${pad(now.getMinutes())}</span> ${msg}`;
    }
    appendMessages(msg);
}

function handleChatCommand(data) {
    function openMenu(url) {
        if (data.Arguments && data.Arguments.length > 0) {
            url = data.Arguments[0];
        }
        window.open(url, '_blank', 'menubar=0,status=0,toolbar=0,width=300,height=600');
    }

    switch (data.Command) {
        case CommandType.CmdPlaying:
            if (!data.Arguments) {
                setPlaying('', '');
            } else if (data.Arguments.length == 1) {
                setPlaying(data.Arguments[0], '');
            } else {
                setPlaying(data.Arguments[0], data.Arguments[1]);
            }
            break;
        case CommandType.CmdRefreshPlayer:
            // calling a video function
            if (typeof initPlayer !== 'undefined') {
                initPlayer();
            }
            break;
        case CommandType.CmdPurgeChat:
            purgeChat();
            appendMessages(`<span class="notice">Chat has been purged by a moderator.</span>`);
            break;
        case CommandType.CmdHelp:
            openMenu('/help');
            break;
        case CommandType.CmdEmotes:
            openMenu('/emotes');
            break;
    }
}

/**
 * @param {*} message
 */
function recieveMessage(message) {
    switch (message.Type) {
        case DataType.DTHidden:
            handleHiddenMessage(message.Data);
            break;
        case DataType.DTEvent:
            if (message.Data.Event != EventType.EvServerMessage) {
                sendMessage('', ClientDataType.CdUsers);
            }
        case DataType.DTChat:
            handleChatMessage(message.Data, message.Type == DataType.DTEvent);
            break;
        case DataType.DTCommand:
            handleChatCommand(message.Data);
            break;
        default:
            break;
    }
}

/**
 * @param {string} data
 * @param {boolean} log
 */
function websocketSend(data, log = true) {
    if (log) {
        console.log(data);
    }
    if (ws.readyState == ws.OPEN) {
        ws.send(data);
    } else {
        console.log('did not send data because websocket is not open', data);
    }
}

/**
 * @param {string|any} msg
 * @param {number} type
 * @param {boolean} log
 */
function sendMessage(msg, type, log = true) {
    if (typeof msg !== 'string') {
        msg = JSON.stringify(msg);
    }

    if (!type) {
        type = ClientDataType.CdMessage;
    }

    websocketSend(JSON.stringify({
        Type: type,
        Message: msg,
    }), log);
}

function sendChat() {
    sendMessage($('#msg').val());
    $('#msg').val('');
}

function emoteToHtml(file, title) {
    return `<img src="${file}" class="emote" title="${title}" />`
}

function updateSuggestionCss(m) {
    if ($('#suggestions').children().length > 0) {
        $('#suggestions').css('bottom', `${$('#msg').outerHeight(true) - 1}px`);
        $('#suggestions').css('display', '');
    } else {
        $('#suggestions').css('display', 'none');
    }
}

function updateSuggestionScroll() {
    let item = $('#suggestions .selectedName');
    if (item.length !== 0) {
        item[0].scrollIntoView({ block: 'center' });
    }
}

function updateSuggestionDiv() {
    const selectedClass = ` class="selectedName"`;

    let divs = Array(filteredSuggestion.length);
    if (filteredSuggestion.length > 0) {
        if (currentSuggestion == '') {
            currentSuggestion = filteredSuggestion[filteredSuggestion.length - 1]
        }

        let hasCurrentSuggestion = false;
        for (let i = 0; i < filteredSuggestion.length; i++) {
            divs[i] = '<div';
            let suggestion = filteredSuggestion[i];
            if (suggestion == currentSuggestion) {
                hasCurrentSuggestion = true;
                divs[i] += selectedClass;
            }
            divs[i] += ` onclick="clickEmote(${i})"`;
            divs[i] += '>';

            if (currentSuggestionType == SuggestionType.Emote) {
                divs[i] += emoteToHtml(emotes[suggestion], suggestion);
            }

            divs[i] += `${suggestion}</div>`;
        }

        if (!hasCurrentSuggestion) {
            divs[0] = divs[0].slice(0, 4) + selectedClass + divs[0].slice(4);
        }
    }
    $('#suggestions')[0].innerHTML = divs.join('\n');
    updateSuggestionScroll();
}

function clickEmote(idx) {
    currentSuggestion = filteredSuggestion[idx];
    processMessageKey({ keyCode: 13, ctrlKey: false })
}

function processMessageKey(e) {
    let keyCode = e.keyCode;
    let ctrl = e.ctrlKey;

    // ctrl + space
    if (ctrl && keyCode == 32) {
        processMessage();
        return true;
    }

    if (filteredSuggestion.length == 0 || currentSuggestion == '') {
        return false;
    }

    switch (keyCode) {
        case 27: // esc
            filteredSuggestion = [];
            currentSuggestion = '';
            currentSuggestionType = SuggestionType.None;
            break;
        case 38: // up
        case 40: // down
            let newIdx = 0;
            for (let i = 0; i < filteredSuggestion.length; i++) {
                const n = filteredSuggestion[i];
                if (n == currentSuggestion) {
                    newIdx = i;
                    if (keyCode == 40) {
                        newIdx = i + 1;
                        if (newIdx == filteredSuggestion.length) {
                            newIdx--;
                        }
                    } else if (keyCode == 38) {
                        newIdx = i - 1;
                        if (newIdx < 0) {
                            newIdx = 0;
                        }
                    }
                    break;
                }
            }
            currentSuggestion = filteredSuggestion[newIdx];
            break;
        case 9: // tab
        case 13: // enter
            const re = /[:@]([\w]+|$)(\s|$)/;

            let replaceVal = '';
            if (currentSuggestionType == SuggestionType.Emote) {
                replaceVal = `:${currentSuggestion}:`;
            } else {
                replaceVal = `@${currentSuggestion}`;
            }

            let msg = $('#msg');
            let val = msg.val();

            let match = val.match(re);
            let endsSpace = match[0].endsWith(' ');
            if (endsSpace) {
                replaceVal += ' ';
            }

            let idx = val.indexOf(match[0]) + replaceVal.length;
            let newVal = val.replace(re, replaceVal);

            msg.val(newVal);
            msg[0].selectionStart = idx;
            msg[0].selectionEnd = idx;

            filteredSuggestion = [];
            break;
        default:
            return false;
    }

    updateSuggestionDiv();
    return true;
}

function processMessage() {
    function handleSuggestion(msg, cmp) {
        if (msg.length == 1 || cmp.toLowerCase().startsWith(msg.slice(1))) {
            filteredSuggestion.push(cmp);
        }
    }

    let text = $('#msg').val().toLowerCase();
    let startIdx = $('#msg')[0].selectionStart;

    filteredSuggestion = [];
    if (text && (users || emotes)) {
        let parts = text.split(' ')

        let caret = 0;
        for (let i = 0; i < parts.length; i++) {
            const word = parts[i];
            // Increase caret index at beginning if not first word to account for spaces
            if (i != 0) {
                caret++;
            }

            // It is possible to have a double space "  ", which will lead to an
            // empty string element in the slice. Also check that the index of the
            // cursor is between the start of the word and the end
            if (word && caret <= startIdx && startIdx <= caret + word.length) {
                if (word[0] == '@') {
                    currentSuggestionType = SuggestionType.Name;
                    users.forEach(name => handleSuggestion(word, name));
                } else if (word[0] == ':') {
                    currentSuggestionType = SuggestionType.Emote;
                    Object.keys(emotes).forEach(emote => handleSuggestion(word, emote));
                }
            }

            if (filteredSuggestion.length > 0) {
                currentSuggestion = '';
                break;
            }

            caret += word.length;
        }
    }

    updateSuggestionDiv();
}

/**
 * @param {string} msg
 */
function setNotifyBox(msg = '') {
    $('#notifyBox').html(msg);
}

// Button Wrapper Functions
function auth() {
    let pass = prompt('Enter pass');
    if (pass != '' && pass !== null) {
        sendMessage(`/auth ${pass}`);
    }
}

function nick() {
    let nick = prompt('Enter new name');
    if (nick != '' && nick !== null) {
        sendMessage(`/nick ${nick}`);
    }
}

function help() {
    sendMessage('/help');
}

function showColors(show) {
    if (show === undefined) {
        show = $('#hiddencolor').css('display') === 'none';
    }

    $('#hiddencolor').css('display', show ? 'block' : '');
}

function colorAsHex() {
    let r = parseInt($('#colorRed').val()).toString(16).padStart(2, '0');
    let g = parseInt($('#colorGreen').val()).toString(16).padStart(2, '0');
    let b = parseInt($('#colorBlue').val()).toString(16).padStart(2, '0');
    return `#${r}${g}${b}`
}

function updateColor() {
    let r = $('#colorRed').val();
    let g = $('#colorGreen').val();
    let b = $('#colorBlue').val();

    $('#colorRedLabel').text(r.padStart(3, '0'));
    $('#colorGreenLabel').text(g.padStart(3, '0'));
    $('#colorBlueLabel').text(b.padStart(3, '0'));

    $('#colorName').css('color', `rgb(${r}, ${g}, ${b})`);

    if (isValidColor(colorAsHex())) {
        $('#colorWarning').text('');
    } else {
        $('#colorWarning').text('Unreadable Color');
    }
}

function changeColor() {
    if (isValidColor(colorAsHex())) {
        sendColor(colorAsHex());
    }
}

function colorSelectChange() {
    let val = $('#colorSelect').val()
    if (val !== '') {
        sendColor(val);
    }
}

/**
 * @param {string} color
 */
function sendColor(color) {
    sendMessage(`/color ${color}`);
    showColors(false);
}

// Get the websocket setup in a function so it can be recalled
function setupWebSocket() {
    ws = new WebSocket(getWsUri());
    ws.onmessage = (m) => recieveMessage(JSON.parse(m.data));
    ws.onopen = () => {
        console.log('Websocket Open');
        // Ngnix websocket timeouts at 60s
        // http://nginx.org/en/docs/http/websocket.html
        setInterval(() => { sendMessage('', ClientDataType.CdPing, false) }, 45000);
    }
    ws.onclose = () => {
        closeChat();
        setNotifyBox('The connection to the server has closed. Please refresh page to connect again.');
        $('#joinbox').css('display', 'none');
    }
    ws.onerror = (e) => {
        console.log('Websocket Error:', e);
        e.target.close();
    }
}

function setupEvents() {
    $('#name').on({
        keypress: (e) => {
            if (e.originalEvent.keyCode == 13) {
                $('#join').trigger('click');
            }
        }
    });

    $('#msg').on({
        keypress: (e) => {
            if (e.originalEvent.keyCode == 13 && !e.originalEvent.shiftKey) {
                $('#send').trigger('click');
                e.preventDefault();
            }
        },
        keydown: (e) => {
            if (processMessageKey(e)) {
                e.preventDefault();
            }
        },
        input: () => processMessage(),
    });

    $('#send').on({
        click: () => $('#msg').focus(),
    });

    var suggestionObserver = new MutationObserver(
        (mutations) => mutations.forEach(updateSuggestionCss)
    ).observe($('#suggestions')[0], { childList: true });
}

function join() {
    color = getCookie('color');
    if (!color) {
        // If color is not set then we want to assign a random color to the user
        color = randomColor();
    } else if (!isValidColor(color)) {
        console.info(`${color} is not a valid color, clearing cookie`);
        deleteCookie('color');
    }

    sendMessage({
        Name: $('#name').val(),
        Color: color,
    }, ClientDataType.CdJoin);
}

window.addEventListener('onresize', updateSuggestionCss);

window.addEventListener('load', () => {
    setNotifyBox();
    setupWebSocket();
    setupEvents();

    // Make sure name is focused on start
    $('#name').focus();
    $('#timestamp').prop('checked', getCookie('timestamp') === 'true');
});
