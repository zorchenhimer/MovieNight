/// <reference path="./jquery.js" />

// Make this on all pages so video page also doesn't do this
$(document).on("keydown", function (e) {
    if (e.which === 8 && !$(e.target).is("input, textarea")) {
        e.preventDefault();
    }
});
