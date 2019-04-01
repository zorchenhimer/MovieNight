/// <reference path="./jquery.js" />

let konamiCode = ["ArrowUp", "ArrowUp", "ArrowDown", "ArrowDown", "ArrowLeft", "ArrowRight", "ArrowLeft", "ArrowRight", "b", "a"]
let lastKeys = []
let devKeys = false;

// Make this on all pages so video page also doesn't do this
$(document).on("keydown", function (e) {
    lastKeys.push(e);
    if (lastKeys.length > 10) {
        lastKeys.shift();
    }
    a = e

    if (devKeys) {
        let modifiedLastKeys = []
        lastKeys.forEach((e) => {
            switch (e.key) {
                case " ":
                    modifiedLastKeys.push(`Space - ${e.keyCode}`);
                    break;
                default:
                    modifiedLastKeys.push(`${e.key} - ${e.keyCode}`);
                    break;
            }
        })
        $("#devKeys").html(`'${modifiedLastKeys.join("', '")}'`);
    }

    if (e.which === 8 && !$(e.target).is("input, textarea")) {
        e.preventDefault();
    }

    checkKonami(e);
});


function checkKonami(e) {
    if (lastKeys.length === konamiCode.length) {
        for (let i = 0; i < lastKeys.length; i++) {
            if (lastKeys[i] != konamiCode[i]) {
                return;
            }
        }
        $("#remote").css("display", "block");
    }
}

function flipRemote() {
    $("#remote").attr("src", "/static/img/remote_active.png");
    setTimeout(() => {
        $("#remote").attr("src", "/static/img/remote.png");
    }, Math.round(Math.random() * 10000) + 1000);
}

function enableDebug() {
    devKeys = true;
    $("#devKeys").css("display", "block");
}

/*
// Just add a / above to uncomment the block
setTimeout(() => {
    enableDebug();
    alert("Comment this out. It shows the keys.");
}, 150);
//*/
