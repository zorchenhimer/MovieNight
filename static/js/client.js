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
        $('#playingDiv').hide();
        document.title = "Movie Night"
        return;
    }

    $('#playingDiv').show();
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

function appendMessages(msg) {
    $("#messages").append(msg).scrollTop(9e6);
}

function openChat() {
    console.log("chat opening");
    $("#phase1").animate({ opacity: 0 }, 500, "linear", function () {
        $("#phase1").css({ display: "none" })
        $("#phase2").css({ opacity: 1 })
        $("#msg").focus()
    })
}

function closeChat() {
    console.log("chat closing")
    $("#phase1").stop().css({ display: "block" }).animate({ opacity: 1 }, 500)
    $("#phase2").stop().animate({ opacity: 0 })
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

function onloadChat() {
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

    $("#send").click(function () {
        sendMessage($("#msg").val());
        $("#msg").val("");
    })
}