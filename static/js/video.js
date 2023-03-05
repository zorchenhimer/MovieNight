/// <reference path='./both.js' />


function initPlayer() {
    var masterUrl = `${window.location.origin}/master.m3u8`; 

    if (navigator.userAgent.match(/(iPhone|iPod|iPad)/i)) {
        var videoElement = document.getElementById("videoElement");
        videoElement.src = masterUrl;
        videoElement.autoplay = true;
    } else if(Hls.isSupported()) {
        var videoElement = document.getElementById("videoElement");
        var hls = new Hls();
        hls.loadSource(masterUrl);
        hls.attachMedia(videoElement);
        hls.on(Hls.Events.MANIFEST_PARSED,function() {
          videoElement.play();
        });
    }



    if (!flvjs.isSupported()) {
        console.warn('flvjs not supported');
        return;
    }

    let videoElement = document.querySelector('#videoElement');
    let flvPlayer = flvjs.createPlayer({
        type: 'flv',
        url: '/live'
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

window.addEventListener('load', initPlayer);
