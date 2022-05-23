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
function showFiles(drive) {

    fetch(`/drive/${drive}/list`, {
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
    div.innerHTML = `drive ${drive}: ` + data.trim();

    var bt = document.getElementById('btSave');
    bt.name = drive;
    bt.disabled = false;
    bt = document.getElementById('btUnload');
    bt.name = drive;
    bt.disabled = false;
}

//
function operateDrive(drive, action) {

    var path = `/drive/${drive}`;

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
