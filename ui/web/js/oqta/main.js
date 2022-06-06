/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2021, Alexander Vollschwitz

   This file is part of OqtaDrive.

   OqtaDrive is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   OqtaDrive is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with OqtaDrive. If not, see <http://www.gnu.org/licenses/>.
*/

//
var statusIcons = {
    'empty':          'bi-none',
    'idle':           'bi-app',
    'busy':           'bi-caret-right-square',
    'hardware':       'bi-gear',
    'unformatted':    'bi-hr',
    'writeProtected': 'bi-lock',
    'modified':       'bi-app-indicator',
    'connected':      'bi-plug-fill',
    'disconnected':   'bi-plug',
    'loading':        'bi-hourglass-split',
    'locked':         'bi-lock',
    'unlocked':       'bi-unlock',
};

//
function getStatusIcon(s) {
    return statusIcons[s];
}

//
async function subscribe() {

    let response = await fetch('/watch', {
        headers: {
            'Content-Type': 'application/json'
        }
    });

    switch (response.status) {
        case 502:
            break;
        case 200:
            let data = await response.json();
            update(data);
            break;
        default:
            console.log(response.statusText);
            await new Promise(resolve => setTimeout(resolve, 1000));
            break;
    }

    await subscribe();
}

//
function getFormatCompressor(file) {
    var compressor = getCompressor(file);
    if (compressor != '') {
        file = removeExtension(file);
    }
    var format = getFormat(file);
    return {
        "format": format,
        "compressor": compressor
    };
}

//
function getFormat(file) {
    var ext = getExtension(file);
    switch (ext) {
        case 'mdv':
        case 'mdr':
        case 'z80':
        case 'sna':
            return ext;
    }
    return '';
}

//
function getCompressor(file) {
    var ext = getExtension(file);
    switch (ext) {
        case 'gz':
        case 'gzip':
        case 'zip':
        case '7z':
            return ext;
    }
    return '';
}

//
function getExtension(file) {
    var lastDot = file.lastIndexOf('.');
    return file.substring(lastDot + 1).toLowerCase();
}

//
function removeExtension(file) {
    var lastDot = file.lastIndexOf('.');
    return file.substring(0, lastDot);
}

//
function getName(path) {
    var lastSlash = path.lastIndexOf('/');
    var name = path.substring(lastSlash + 1);
    var firstDot = name.indexOf('.');
    return name.substring(0, firstDot);
}

//
function userConfirm(title, question, callback) {

    var mod = document.getElementById('modal-confirm');
    mod.querySelector('.modal-title').textContent = title;
    var body = mod.querySelector('.modal-body');
    body.textContent = question;
    body.align = "left";

    var modInst = new bootstrap.Modal(mod, null);

    mod.querySelector('.btn-primary').onclick = function() {
        modInst.hide();
        callback(true);
    };
    mod.querySelector('.btn-secondary').onclick = function() {
        modInst.hide();
        callback(false);
    };

    modInst.show();
}

//
function userAlert(title, message) {

    var mod = document.getElementById('modal-alert');
    mod.querySelector('.modal-title').textContent = title;

    var body = mod.querySelector('.modal-body');
    body.textContent = message;
    body.align = "left";

    var modInst = new bootstrap.Modal(mod, null);

    mod.querySelector('.btn-primary').onclick = function() {
        modInst.hide();
    };

    modInst.show();
    return modInst;
}

//
async function showTab(t) {
    var triggerEl = document.querySelector(`a[data-bs-target="#${t}"]`);
    bootstrap.Tab.getOrCreateInstance(triggerEl).show();
}

// ----------------------------------------------------------------------------

//
var popoverTriggerList = [].slice.call(
    document.querySelectorAll('[data-bs-toggle="popover"]'))
var popoverList = popoverTriggerList.map(function (popoverTriggerEl) {
    return new bootstrap.Popover(popoverTriggerEl)
})

//
var selectedSearchItem = "";

// FIXME make more compact

//
fetch('/list', {
    headers: {
        'Content-Type': 'application/json'
    }
}).then(
    response => response.json()
).then(
    data => buildList(data)
).catch(
    err => console.log('error: ' + err)
);

//
fetch('/status', {
    headers: {
        'Content-Type': 'application/json'
    }
}).then(
    response => response.json()
).then(
    data => updateClient(data.client)
).catch(
    err => console.log('error: ' + err)
);

//
document.getElementById('btClient').onclick = function() {
    resetClient();
};

var bt = document.getElementById('btSave');
bt.onclick = function() {
    operateDrive(this.name, 'save');
};
bt.disabled = true;

bt = document.getElementById('btUnload');
bt.onclick = function() {
    operateDrive(this.name, 'unload');
};
bt.disabled = true;

//
getVersion();
getRumbleLevel();
setupSearch();
setupConfig();
subscribe();
