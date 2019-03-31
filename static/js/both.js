/// <reference path="./jquery.js" />

let konamiCode = ["ArrowUp", "ArrowUp", "ArrowDown", "ArrowDown", "ArrowLeft", "ArrowRight", "ArrowLeft", "ArrowRight", "b", "a"]
let lastKeys = []

// Make this on all pages so video page also doesn't do this
$(document).on("keydown", function (e) {
    checkKonami(e);

    if (e.which === 8 && !$(e.target).is("input, textarea")) {
        e.preventDefault();
    }
});


function checkKonami(e) {
    lastKeys.push(e.key);
    if (lastKeys.length > 10) {
        lastKeys.shift();
    }

    if (lastKeys.length === konamiCode.length) {
        for (let i = 0; i < lastKeys.length; i++) {
            if (lastKeys[i] != konamiCode[i]) {
                return;
            }
        }
        $("#remote").css("display", "");
    }
}

function flipRemote() {
    $("#remote").attr("src", "/static/img/remote_active.png");
    setTimeout(() => {
        $("#remote").attr("src", "/static/img/remote.png");
    }, Math.round(Math.random() * 10000) + 1000);
}
