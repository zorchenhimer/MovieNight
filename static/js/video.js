/// <reference path="./both.js" />


function initPlayer() {
    if (!flvjs.isSupported()) {
        console.warn('flvjs not supported');
        return;
    }

    let videoElement = document.querySelector("#videoElement");
    let flvPlayer = flvjs.createPlayer({
        type: "flv",
        url: "/live"
    });
    flvPlayer.attachMediaElement(videoElement);
    flvPlayer.load();
    flvPlayer.play();

    let overlay = document.querySelector('#videoOverlay');
    overlay.onclick = () => {
        overlay.style.display = 'none';
        videoElement.muted = false;
    };
}

window.addEventListener("load", initPlayer);
