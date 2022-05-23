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
        fc.accept = '.mdr,.MDR,.mdv,.MDV,.z80,.Z80,.sna,.SNA,.zip,.ZIP,.gz,.GZ,.gzip,.GZIP,.7z,.7Z';
        fc.style = 'display:none;';
        fc.onclick = function() {
            this.value = null;
        };
        fc.onchange = function() {
            var name = this.files[0].name;
            var drive = this.id.substring(2);
            indicateLoading(drive);
            var fc = getFormatCompressor(name);
            upload(drive, getName(name), fc.format, fc.compressor, this.files[0], false);
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

    document.getElementById('clientIcon').className = getStatusIcon(s);
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
function indicateLoading(drive) {
    document.getElementById('bt' + drive).disabled = true;
    document.getElementById('name' + drive).innerHTML = '&lt; loading &gt;';
    document.getElementById('it' + drive).className = getStatusIcon('loading');
}

//
function upload(drive, name, format, compressor, data, isRef) {

    var path = `/drive/${drive}?type=${format}&compressor=${compressor}&repair=true&name=`
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

    it.className = getStatusIcon(s);
}
