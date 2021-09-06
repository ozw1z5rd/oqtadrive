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
    'disconnected':   'bi-plug'
};

//
function buildList(drives) {

    var driveList = document.getElementById('driveList');

    for (var i = 1; i <= drives.length; i++) {

        var form = document.createElement('form');
        driveList.appendChild(form);

        var row = document.createElement('div');
        row.className = 'row';
        form.appendChild(row);

        var fc = document.createElement('input');
        fc.className = 'custom-file-input';
        fc.id = 'fc' + i;
        fc.type = 'file';
        fc.accept = '.mdr,.MDR,.mdv,.MDV,.z80,.Z80';
        fc.style = 'display:none;';
        fc.onclick = function() {
            this.value = null;
        };
        fc.onchange = function() {
            var name = this.files[0].name;
            upload(this.id.substring(2), getName(name), getFormat(name),
                this.files[0], false);
        };
        row.appendChild(fc);

        var div = document.createElement('div');
        div.className = 'col-2';

        var bt = document.createElement('input');
        bt.className = 'btn btn-outline-light mb-1';
        bt.id = 'bt' + i;
        bt.type = 'button';
        bt.value = i;
        bt.onclick = function() {
            var drive = this.id.substring(2);
            if (!loadSelectedSearchItem(drive)) {
                document.getElementById('fc' + drive).click();
            }
        };
        configureButton(bt, drives[i-1]);
        div.appendChild(bt);
        row.appendChild(div);

        div = document.createElement('div');
        div.id = 'name' + i;
        div.className = 'col-7';
        div.align = 'left';
        setName(div, drives[i-1]);
        div.onclick = function() {
            showFiles(this.id.substring(4));
        };
        row.appendChild(div);

        div = document.createElement('div');
        div.className = 'col-3';
        var it = document.createElement('i');
        it.id = 'it' + i;
        setStatusIcon(it, drives[i-1]);
        div.appendChild(it);
        row.appendChild(div);
    }
}

//
function update(data) {
    updateClient(data.client);
    updateList(data.drives);
}

//
function updateClient(cl) {

    if (cl == "") {
        return;
    }

    var s = "connected";
    if (cl == '<unknown>') {
        s = 'disconnected';
        cl = 'disconn.';
    }

    document.getElementById('clientIcon').className = statusIcons[s]
    document.getElementById('clientLabel').innerHTML = cl;
}

//
function updateList(drives) {

    if (drives == null) {
        return;
    }

    for (var i = 1; i <= drives.length; i++) {
        var d = drives[i-1];
        setStatusIcon(document.getElementById('it' + i), d);
        setName(document.getElementById('name' + i), d);
        configureButton(document.getElementById('bt' + i), d);
    }
}

//
function showFiles(drive) {

    fetch('/drive/' + drive + '/list', {
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(
        response => response.text()
    ).then(
        data => updateFileList(drive, data)
    ).catch(
        err => console.log('error: ' + err)
    );

    showTab('files');
}

//
function updateFileList(drive, data) {

    var div = document.getElementById('fileList');
    div.innerHTML = 'drive ' + drive + ': ' + data.trim();

    var bt = document.getElementById('btSave');
    bt.name = drive;
    bt.disabled = false;
    bt = document.getElementById('btUnload');
    bt.name = drive;
    bt.disabled = false;
}

//
function operateDrive(drive, action) {

    var path = '/drive/' + drive;

    switch (action) {

        case 'save':
            alert("SAVE coming soon");
            return;
            break;

        case 'unload':
            path += '/unload?force=true'
            userConfirm("Unload cartridge?", "Unsaved changes will be lost!",
                function(confirmed) {
                    if (confirmed) {
                        operateDriveDo(drive, action, path);
                    }
                });
            break;

        default:
            return;
    }
}

//
function operateDriveDo(drive, action, path) {

    fetch(path, {
        headers: {
            'Content-Type': 'application/json'
        }
    }).catch(
        err => console.log('error: ' + err)
    );

    if (action == 'unload') {
        showTab('drives');
    }
}

//
function configureButton(bt, data) {
    bt.disabled = (data.status == 'busy' || data.status == 'hardware');
}

//
function setName(div, data) {
    if (data.name != "" || data.status != 'busy') {
        if (data.formatted) {
            div.innerHTML = data.name;
        } else if (data.status == 'hardware') {
            div.innerHTML = '&lt; h/w drive &gt;';
        } else {
            div.innerHTML = '&lt; unformatted &gt;';
        }
    }
}

//
function setStatusIcon(it, data) {

    var s = data.status;

    if (data.status == 'idle') {
        if (data.modified) {
            s = 'modified';
        } else if (data.writeProtected) {
            s = 'writeProtected';
        } else if (!data.formatted) {
            s = 'unformatted';
        }
    }

    it.className = statusIcons[s];
}

//
function upload(drive, name, format, data, isRef) {

    var path = '/drive/' + drive + '?type=' + format + '&repair=true&name='
        + encodeURIComponent(name);

    if (isRef) {
        path += "&ref=true"
    }

    fetch(path, {
        method: 'PUT',
        body: data
    }).then(
        response => response.json()
    ).then(
        success => console.log(success)
    ).catch(
        err => console.log('error: ' + err)
    );
};

//
function resetClient() {
    fetch('/resync?reset=true', {method: 'PUT'}).then(
        function(){}
    ).then(
        success => console.log(success)
    ).catch(
        err => console.log('error: ' + err)
    );
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
function setupSearch() {

    var keyword = document.getElementById('repo-search');
    keyword.addEventListener('keyup', function() {
        keyupStack.push(1);
        setTimeout(function() {
            keyupStack.pop();
            if (keyupStack.length === 0) {
                search(this.value);
            }
        }.bind(this), 600);
    });
}

//
function search(term) {

    fetch('/search?items=25&term=' + term, {
        headers: {
            'Content-Type': 'text'
        }
    }).then(
        response => response.text()
    ).then(
        data => updateSearchResults(data)
    ).catch(
        err => console.log('error: ' + err)
    );
}

//
function updateSearchResults(data) {

    var list = document.getElementById('search-results');
    list.textContent = null;

    data.split("\n").forEach(function(l) {
        if (l == "") {
            return;
        }
        var li = document.createElement('li');
        li.className = "list-group-item text-white bg-dark";
        li.align = "left";
        li.onclick = function() {
            searchItemSelected(this.innerHTML);
        }
        li.appendChild(document.createTextNode(l));
        list.appendChild(li);
    });
}

//
function searchItemSelected(item) {
    userConfirm("Load cartridge?",
        "Confirm & click the load button of the drive into which you want to load.",
        function(confirmed) {
            if (confirmed) {
                selectedSearchItem = "repo://" + item;
                showTab('drives');
            } else {
                selectedSearchItem = "";
            }
        });
}

//
function loadSelectedSearchItem(drive) {
    if (selectedSearchItem == "") {
        return false;
    }
    upload(drive, getName(selectedSearchItem), getFormat(selectedSearchItem),
        selectedSearchItem, true);
    selectedSearchItem = "";
    return true;
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

var selectedSearchItem = "";

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

var keyupStack = [];
setupSearch()

subscribe();
