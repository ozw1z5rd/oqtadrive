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
    'loading':        'bi-hourglass-split'
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
function getFormat(file) {
    var lastDot = file.lastIndexOf('.');
    return file.substring(lastDot + 1);
}

//
function getName(path) {
    var lastSlash = path.lastIndexOf('/');
    var name = path.substring(lastSlash + 1);
    var lastDot = name.lastIndexOf('.');
    return name.substring(0, lastDot);
}

//
function userConfirm(title, question, callback) {

    var mod = document.getElementById('modal');
    mod.querySelector('.modal-title').textContent = title;
    mod.querySelector('.modal-body').textContent = question;

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
async function showTab(t) {
    var triggerEl = document.querySelector('a[data-bs-target="#' + t + '"]');
    bootstrap.Tab.getOrCreateInstance(triggerEl).show();
}

// ----------------------------------------------------------------------------

//
var popoverTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="popover"]'))
var popoverList = popoverTriggerList.map(function (popoverTriggerEl) {
    return new bootstrap.Popover(popoverTriggerEl)
})

//
var selectedSearchItem = "";

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
setupSearch()
subscribe();
