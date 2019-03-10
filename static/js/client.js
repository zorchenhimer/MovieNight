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

function escapeHtml(string) {
    return string;
    //return String(string).replace(/[&<>"'\/]/g, function (s) {
    //    return entityMap[s];
    //});
}

//helper function for debugging purposes
function toHex(str) {
    var result = '';
    for (var i = 0; i < str.length; i++) {
        result += ("0" + str.charCodeAt(i).toString(16)).slice(-2) + " ";
    }
    return result.toUpperCase();
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

window.onload = function () {
    console.log("TEST 1")
    $("INPUT").val("")
    $("#name").keypress(function (evt) {
        console.log("TEST 2")
        if (evt.originalEvent.keyCode == 13) {
            console.log("TEST 3")
            $("#join").trigger("click")
        }
    })

    //handling the start of the chat
    $("#join").click(function () {
        $("#error").html("");
        var name = escapeHtml($("#name").val())
        if (name.length < 3) {
            $("#error").html("Name is too short!");
            return
        }
        console.log("join started")
        chat = new WebSocket("ws://" + window.location.host + ":8089/ws");
        chat.onopen = function (evt) {
            chat.send(name);  //sending the chat name
            $("#phase1").animate({ opacity: 0 }, 500, "linear", function () {
                $("#phase1").css({ display: "none" })
                $("#phase2").css({ opacity: 1 })
                $("#msg").focus()
            })
        };
        chat.onerror = function (evt) {
            console.log("Websocket Error:", evt)
        };
        chat.onclose = function (evt) {
            console.log("chat closing")
            $("#phase1").stop().css({ display: "block" }).animate({ opacity: 1 }, 500)
            $("#phase2").stop().animate({ opacity: 0 })
            $("#error").html("That name was already used!")
        };
        chat.onmessage = function (evt) {
            $("#messages").append(evt.data).scrollTop(9e6)
        };

    })

    $("#msg").keypress(function (evt) {
        if (evt.originalEvent.keyCode == 13 && !evt.originalEvent.shiftKey) {
            $("#send").trigger("click")
            evt.preventDefault();
            // submit name
        }
    })

    $("#send").click(function () {
        chat.send(escapeHtml($("#msg").val()));
        $("#msg").val("");
    })

    //helper function for escaping HTML
    var entityMap = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': '&quot;',
        "'": '&#39;',
        "/": '&#x2F;',
        "\n": '<BR/>'
    };
}