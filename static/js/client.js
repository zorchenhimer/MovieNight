function initPlayer() {
    if (flvjs.isSupported()) {
        var videoElement = document.getElementById('videoElement');
        var flvPlayer = flvjs.createPlayer({
            type: 'flv',
            url: '/live'
        });
        flvPlayer.attachMediaElement(videoElement);
        flvPlayer.load();
        flvPlayer.play();
    }
}

function setPlaying(title, link) {
    if (title === "") {
        $('#playing').hide();
        $('#playinglink').hide();
        document.title = "Movie Night"
        return;
    }

    $('#playing').show();
    $('#playing').text(title);
    document.title = "Movie Night | " + title

    if (link === "") {
        $('#playinglink').hide();
        return;
    }

    $('#playinglink').show();
    $('#playinglink').text('[Info Link]');
    $('#playinglink').attr('href', link);
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
        go.run(result.instance)
    }).catch((err) => {
        console.error(err);
    });
}

function getWsUri() {
    port = window.location.port
    if (port == "") {
        port = "8089"
    }
    return "ws://" + window.location.hostname + ":" + port + "/ws"
}

let maxMessageCount = 0
function appendMessages(msg) {
    let msgs = $("#messages").find('div')

    // let's just say that if the max count is less than 1, then the count is infinite
    // the server side should take care of chaking max count ranges
    if (msgs.length > maxMessageCount) {
        msgs.first().remove()
    }

    $("#messages").append(msg).scrollTop(9e6);
}

function openChat() {
    console.log("chat opening");
    $("#joinbox").css("display", "none")
    $("#chat").css("display", "grid")
    $("#msg").focus()
}

function closeChat() {
    console.log("chat closing")
    $("#joinbox").css("display", "")
    $("#chat").css("display", "none")
    $("#error").html("That name was already used!")
}

function join() {
    let name = $("#name").val();
    if (name.length < 3 || name.length > 36) {
        $("#error").html("Please input a name between 3 and 36 characters");
        return;
    }
    sendMessage($("#name").val());
    openChat();
}

let ws = new WebSocket(getWsUri());
ws.onmessage = (m) => recieveMessage(m.data);
ws.onopen = (e) => console.log("Websocket Open:", e);
ws.onclose = () => closeChat();
ws.onerror = (e) => console.log("Websocket Error:", e);

function websocketSend(data) {
    ws.send(data)
}

function sendChat() {
    sendMessage($("#msg").val());
    $("#msg").val("");
}


function chatOnload() {
    startGo();

    $("#name").keypress(function (evt) {
        if (evt.originalEvent.keyCode == 13) {
            $("#join").trigger("click")
        }
    })

    $("#msg").keypress(function (evt) {
        if (evt.originalEvent.keyCode == 13 && !evt.originalEvent.shiftKey) {
            $("#send").trigger("click")
            evt.preventDefault();
        }
    })
}
