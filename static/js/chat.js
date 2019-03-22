/// <reference path="./both.js" />

function setPlaying(title, link) {
    if (title !== "") {
        $('#playing').text(title);
        document.title = "Movie Night | " + title;
    }

    $('#playing').removeAttr('href');
    if (link !== "") {
        $('#playing').attr('href', link);
    }
}

function startGo() {
    if (!WebAssembly.instantiateStreaming) { // polyfill
        WebAssembly.instantiateStreaming = async (resp, importObject) => {
            const source = await (await resp).arrayBuffer();
            return await WebAssembly.instantiate(source, importObject);
        };
    }

    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/static/main.wasm"), go.importObject).then((result) => {
        go.run(result.instance);
    }).catch((err) => {
        console.error(err);
    });
}

function getWsUri() {
    port = window.location.port;
    if (port != "") {
        port = ":" + port;
    }
    return "ws://" + window.location.hostname + port + "/ws";
}

let maxMessageCount = 0
function appendMessages(msg) {
    let msgs = $("#messages").find('div');

    // let's just say that if the max count is less than 1, then the count is infinite
    // the server side should take care of chaking max count ranges
    if (msgs.length > maxMessageCount) {
        msgs.first().remove();
    }

    $("#messages").append(msg);
    $("#messages").children().last()[0].scrollIntoView({ block: "end", behavior: "smooth" });
}

function purgeChat() {
    $('#messages').empty()
}

inChat = false
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

function join() {
    let name = $("#name").val();
    if (!isValidName(name)) {
        setNotifyBox("Please input a valid name");
        return;
    }
    if (!sendMessage($("#name").val())) {
        setNotifyBox("could not join");
        return;
    }
    setNotifyBox();
    openChat();
}

function websocketSend(data) {
    ws.send(data);
}

function sendChat() {
    sendMessage($("#msg").val());
    $("#msg").val("");
}

function updateSuggestionCss(m) {
    if ($("#suggestions").children().length > 0) {
        $("#suggestions").css("bottom", $("#msg").outerHeight(true) - 1 + "px");
    }
}

function setNotifyBox(msg = "") {
    $("#notifyBox").html(msg);
}

// Button Wrapper Functions
function auth() {
    let pass = prompt("pass please")
    if (pass != "") {
        sendMessage("/auth " + pass);
    }
}

function help() {
    sendMessage("/help")
}

// Get the websocket setup in a function so it can be recalled
function setupWebSocket() {
    ws = new WebSocket(getWsUri());
    ws.onmessage = (m) => recieveMessage(m.data);
    ws.onopen = (e) => console.log("Websocket Open:", e);
    ws.onclose = () => closeChat();
    ws.onerror = (e) => console.log("Websocket Error:", e);
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

    $("#hiddenColorPicker").on({
        change: () => sendMessage("/color " + $("#hiddenColorPicker").val()),
    });

    $("#send").on({
        click: () => $("#msg").focus(),
    });

    var suggestionObserver = new MutationObserver(
        (mutations) => mutations.forEach(updateSuggestionCss)
    ).observe($("#suggestions")[0], { childList: true });
}

window.addEventListener("onresize", updateSuggestionCss);

window.addEventListener("load", () => {
    setNotifyBox();
    setupWebSocket();
    startGo();
    setupEvents();

    // Make sure name is focused on start
    $("#name").focus();
});
