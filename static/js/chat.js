/// <reference path="./both.js" />
/// <reference path="./consts.js" />

/*
processMessageKey
processMessage
*/


let maxMessageCount = 0
let inChat = false;
let users = []
let emotes = {}

function debug() {
    let color = getCookie("color");
    let timestamp = getCookie("timestamp");

    Object.entries({
        maxMessageCount,
        inChat,
        users,
        emotes,
        color,
        timestamp,
    }).forEach(([k, v]) => {
        console.log(k, v);
    });
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
    color = color.replace(/^#/, "", color).toLowerCase();
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
    if (title !== "") {
        $('#playing').text(title);
        document.title = pageTitle + " | " + title;
    } else {
        $('#playing').text("");
        document.title = pageTitle;
    }

    $('#playing').removeAttr('href');
    if (link !== "") {
        $('#playing').attr('href', link);
    }
}

function getWsUri() {
    port = window.location.port;
    if (port != "") {
        port = ":" + port;
    }
    proto = "ws://"
    if (location.protocol == "https:") {
        proto = "wss://"
    }
    return proto + window.location.hostname + port + "/ws";
}

/**
 * @param {string} msg
 */
function appendMessages(msg) {
    let msgs = $("#messages").find('div');

    // let's just say that if the max count is less than 1, then the count is infinite
    // the server side should take care of chaking max count ranges
    if (msgs.length > maxMessageCount) {
        msgs.first().remove();
    }

    $("#messages").append(`<div>${msg}</div>`);
    $("#messages").children().last()[0].scrollIntoView({ block: "end" });
}

function purgeChat() {
    $('#messages').empty()
}

function openChat() {
    console.log("chat opening");
    $("#joinbox").css("display", "none");
    $("#chat").css("display", "grid");
    $("#hidden").css("display", "")
    $("#msg").val("");
    $("#msg").focus();
    inChat = true;
}

function closeChat() {
    console.log("chat closing");
    $("#joinbox").css("display", "");
    $("#chat").css("display", "none");
    $("#hidden").css("display", "none")
    setNotifyBox("That name was already used!");
    inChat = false;
}

function handleHiddenMessage(data) {
    switch (data.Type) {
        case ClientDataType.CdUsers:
            users = data.Data;
            break;
        case ClientDataType.CdColor:
            setCookie("color", data.Data);
            break;
        case ClientDataType.CdEmote:
            emotes = data.Data;
            break;
        case ClientDataType.CdJoin:
            setNotifyBox("");
            openChat();
            break;
        case ClientDataType.CdNotify:
            setNotifyBox(data.Data);
            break;
        default:
            console.warn("unhandled hidden type", data);
            break;
    }
}

/**
 * @param {*} data
 * @param {bool} isEvent
 */
function handleChatMessage(data, isEvent) {
    console.warn(data);
    msg = data.Message;

    if (isEvent) {
        function nameChangeMsg(forced) {
            let users = data.User.split(":");

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

        switch (data.Type) {
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
                nameChangeMsg(false);
                break;
            case EventType.EvNameChangeForced:
                nameChangeMsg(true);
                break;
        }

    } else {
        function spanMsg(className, content) {
            return `<span class="${className}">${content}</span>`;;
        }
        switch (data.Type) {
            case MessageType.MsgAction:
                msg = `<span style="color:${data.Color}">${spanMsg("name", data.From)} ${wrapMsg("cmdme", msg)}</span>`;
                break;
            case MessageType.MsgServer:
                msg = spanMsg("announcement", msg);
                break;
            case MessageType.MsgError:
                msg = spanMsg("error", msg);
                break;
            case MessageType.MsgNotice:
                msg = spanMsg("notice", msg);
                break;
            case MessageType.MsgCommandResponse:
                msg = spanMsg("command", msg);
                break;
            case MessageType.MsgCommandError:
                msg = spanMsg("commanderror", msg);
                break;
            default:
                msg = spanMsg("msg", msg);
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

    if (getCookie("timestamp") === "true" && (data.Type == MessageType.MsgChat || data.Type == MessageType.MsgAction)) {
        let now = new Date();
        let pad = (n) => String(n.toFixed(0)).padStart(2, "0");
        msg = `<span class="time">${pad(now.getHours())}:${pad(now.getMinutes())}</span> ${msg}`;
    }
    appendMessages(msg);
}

function handleChatCommand(data) {
    function openMenu(url) {
        if (data.Arguments && data.Arguments.length > 0) {
            url = data.Arguments[0];
        }
        window.open(url, "_blank", "menubar=0,status=0,toolbar=0,width=300,height=600")
    }

    switch (data.Command) {
        case CommandType.CmdPlaying:
            if (!data.Arguments) {
                setPlaying("", "");
            } else if (data.Arguments.length == 1) {
                setPlaying(data.Arguments[0], "");
            } else {
                setPlaying(data.Arguments[0], data.Arguments[1]);
            }
            break;
        case CommandType.CmdRefreshPlayer:
            // calling a video function
            if (typeof initPlayer !== "undefined") {
                initPlayer();
            }
            break;
        case CommandType.CmdPurgeChat:
            purgeChat();
            appendMessages(`<span class="notice">Chat has been purged by a moderator.</span>`);
            break;
        case CommandType.CmdHelp:
            openMenu("/help");
            break;
        case CommandType.CmdEmotes:
            openMenu("/emotes");
            break;
    }
}


/**
 * @param {*} message
 */
function recieveMessage(message) {


    console.info(message);
    switch (message.Type) {
        case DataType.DTHidden:
            handleHiddenMessage(message.Data);
            break;
        case DataType.DTEvent:
            if (message.Data.Event != EventType.EvServerMessage) {
                sendMessage("", ClientDataType.CdUsers);
            }
        case DataType.DTChat:
            handleChatMessage(message.Data, message.Type == DataType.DTEvent)
        default:
            break;
    }
}

/**
 * @param {string} data
 */
function websocketSend(data) {
    console.log(data)
    if (ws.readyState == ws.OPEN) {
        ws.send(data);
    } else {
        console.log("did not send data because websocket is not open", data);
    }
}

/**
 * @param {string|any} msg
 * @param {number} type
 */
function sendMessage(msg, type) {
    if (typeof msg !== "string") {
        msg = JSON.stringify(msg);
    }

    if (!type) {
        type = ClientDataType.CdMessage;
    }

    websocketSend(JSON.stringify({
        Type: type,
        Message: msg,
    }));
}


function sendChat() {
    sendMessage($("#msg").val());
    $("#msg").val("");
}

function updateSuggestionCss(m) {
    if ($("#suggestions").children().length > 0) {
        $("#suggestions").css("bottom", $("#msg").outerHeight(true) - 1 + "px");
        $("#suggestions").css("display", "");
    } else {
        $("#suggestions").css("display", "none");
    }
}

function updateSuggestionScroll() {
    let item = $("#suggestions .selectedName");
    if (item.length !== 0) {
        item[0].scrollIntoView({ block: "center" });
    }
}

/**
 * @param {string} msg
 */
function setNotifyBox(msg = "") {
    $("#notifyBox").html(msg);
}

// Button Wrapper Functions
function auth() {
    let pass = prompt("Enter pass");
    if (pass != "" && pass !== null) {
        sendMessage("/auth " + pass);
    }
}

function nick() {
    let nick = prompt("Enter new name");
    if (nick != "" && nick !== null) {
        sendMessage("/nick " + nick);
    }
}

function help() {
    sendMessage("/help");
}

function showColors(show) {
    if (show === undefined) {
        show = $("#hiddencolor").css("display") === "none";
    }

    $("#hiddencolor").css("display", show ? "block" : "");
}

function colorAsHex() {
    let r = parseInt($("#colorRed").val()).toString(16).padStart(2, "0");
    let g = parseInt($("#colorGreen").val()).toString(16).padStart(2, "0");
    let b = parseInt($("#colorBlue").val()).toString(16).padStart(2, "0");
    return `#${r}${g}${b}`
}

function updateColor() {
    let r = $("#colorRed").val();
    let g = $("#colorGreen").val();
    let b = $("#colorBlue").val();

    $("#colorRedLabel").text(r.padStart(3, "0"));
    $("#colorGreenLabel").text(g.padStart(3, "0"));
    $("#colorBlueLabel").text(b.padStart(3, "0"));

    $("#colorName").css("color", `rgb(${r}, ${g}, ${b})`);

    if (isValidColor(colorAsHex())) {
        $("#colorWarning").text("");
    } else {
        $("#colorWarning").text("Unreadable Color");
    }
}

function changeColor() {
    if (isValidColor(colorAsHex())) {
        sendColor(colorAsHex());
    }
}

function colorSelectChange() {
    let val = $("#colorSelect").val()
    if (val !== "") {
        sendColor(val);
    }
}

/**
 * @param {string} color
 */
function sendColor(color) {
    sendMessage("/color " + color);
    showColors(false);
}

// Get the websocket setup in a function so it can be recalled
function setupWebSocket() {
    ws = new WebSocket(getWsUri());
    ws.onmessage = (m) => recieveMessage(JSON.parse(m.data));
    ws.onopen = () => console.log("Websocket Open");
    ws.onclose = () => {
        closeChat();
        setNotifyBox("The connection to the server has closed. Please refresh page to connect again.");
        $("#joinbox").css("display", "none");
    }
    ws.onerror = (e) => {
        console.log("Websocket Error:", e);
        e.target.close();
    }
}

function setupEvents() {
    $("#name").on({
        keypress: (e) => {
            if (e.originalEvent.keyCode == 13) {
                $("#join").trigger("click");
            }
        }
    });

    $("#msg").on({
        keypress: (e) => {
            if (e.originalEvent.keyCode == 13 && !e.originalEvent.shiftKey) {
                $("#send").trigger("click");
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

    $("#send").on({
        click: () => $("#msg").focus(),
    });

    var suggestionObserver = new MutationObserver(
        (mutations) => mutations.forEach(updateSuggestionCss)
    ).observe($("#suggestions")[0], { childList: true });
}

function join() {
    color = getCookie("color");
    if (!color) {
        // If color is not set then we want to assign a random color to the user
        color = randomColor();
    } else if (!isValidColor(color)) {
        console.info(`${color} is not a valid color, clearing cookie`);
        deleteCookie("color");
    }

    sendMessage({
        Name: $("#name").val(),
        Color: color,
    }, ClientDataType.CdJoin);
}

window.addEventListener("onresize", updateSuggestionCss);

window.addEventListener("load", () => {
    setNotifyBox();
    setupWebSocket();
    setupEvents();

    // Make sure name is focused on start
    $("#name").focus();
    $("#timestamp").prop("checked", getCookie("timestamp") === "true");
});
