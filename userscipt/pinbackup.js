// ==UserScript==
// @name     pinbackup-scraper
// @include  https://www.pinterest.*
// @require  https://code.jquery.com/jquery-3.4.1.js
// @require  https://gist.github.com/raw/2625891/waitForKeyElements.js
// @grant    GM_addStyle
// @grant    GM_xmlhttpRequest
// @grant    GM_getValue
// @grant    GM_setValue
// ==/UserScript==

// Can be used with Tampermonkey extension in Firefox or Chrome e.g.
// Sends a JSON request with the board URL to the pinbackup REST API
// to scrape the URL submitted.

waitForKeyElements(".Eqh.wYR.zI7.iyn.Hsu", scrape);

function callServer() {
    var data = "{\"url\": \"" + window.location.href + "\"}";
    console.log("JSON request to server: " + data);

    GM_xmlhttpRequest ( {
    method:     'POST',
    url:        'http://127.0.0.1:30000/api/v1/board',
    data:       data,
    headers: {
      "Content-Type": "Content-Type: application/json"
    },
    onload:     function (responseDetails) {
                    console.log (
                        "GM_xmlhttpRequest() response is:\n",
                        responseDetails.responseText
                    );
                }
    } );
}

function scrape(jNode) {
    // Follow button
    $(".tBJ.dyH.iFc.yTZ.erh.tg7.mWe").click(callServer);
}
