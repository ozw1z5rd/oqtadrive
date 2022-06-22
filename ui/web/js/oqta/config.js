/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2022, Alexander Vollschwitz

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
function setupConfig() {
    setupDriveSelect(document.getElementById('map-start'));
    setupDriveSelect(document.getElementById('map-end'));
    getDriveMapping();
}

//
function setupDriveSelect(item) {

    var opt;

    for (var i = 1; i <= 8; i++) {
        opt = document.createElement('option');
        opt.appendChild(document.createTextNode(i));
        item.appendChild(opt);
    }

    opt = document.createElement('option');
    opt.appendChild(document.createTextNode('-'));
    item.appendChild(opt);

    item.addEventListener('change', function() {
        mapDriveSelected(this);
    });
}

//
function updateDriveSelect(data) {

    var start = document.getElementById('map-start');
    start.value = toDriveSelectVal(data.start);
    start.disabled = data.locked;

    var end  = document.getElementById('map-end');
    end.value = toDriveSelectVal(data.end);
    end.disabled = data.locked;

    document.getElementById('btMapSet').disabled = data.locked;

    var s;
    if (data.locked) {
        s = 'locked';
    } else {
        s = 'unlocked';
        mapDriveControl(start, end, true);
    }

    document.getElementById('map-icon').className = getStatusIcon(s);
}

//
function toDriveSelectVal(ix) {
    if (ix < 1) {
        return '-';
    }
    return ix;
}

//
function mapDriveSelected(item) {

    var start = document.getElementById('map-start');
    var end  = document.getElementById('map-end');

    if (item == start) {
        mapDriveControl(start, end, true);
    } else{
        mapDriveControl(end, start, false);
    }
}

//
function mapDriveControl(prim, sec, le) {

    var off = prim.value == '-';

    if (off) {
        sec.value = prim.value;
    } else {
        if (sec.value == '-') {
            sec.value = prim.value;
        }
    }

    for (var i = 1; i <= 8; i++) {
        if (le) {
            prim.options[i-1].disabled = !off && i > sec.value;
            sec.options[i-1].disabled = !off && i < prim.value;
        } else {
            prim.options[i-1].disabled = !off && i < sec.value;
            sec.options[i-1].disabled = !off && i > prim.value;
        }
    }
}

//
function setDriveMapping() {

    var start = document.getElementById('map-start');
    var end  = document.getElementById('map-end');

    if (start.value != '-') {
        userConfirm("Enable hardware drives",
            "Specifying the wrong number of hardware drives will cause problems. If you set too many, you will block virtual drives, if you set too few, the excess hardware drives will conflict with virtual drives, causing bus contention. Proceed?",
            function(confirmed) {
                if (confirmed) {
                    putDriveMapping(start.value, end.value);
                }
            });
    } else {
        userConfirm("Disable hardware drives",
            "Proceed?",
            function(confirmed) {
                if (confirmed) {
                    putDriveMapping(0, 0);
                }
            });
    }
}

//
function getDriveMapping() {
    fetch('/map', {
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(
        response => response.json()
    ).then(
        data => updateDriveSelect(data)
    ).catch(
        err => console.log('error: ' + err)
    );
}

//
function putDriveMapping(start, end) {
    putConfig("Please wait", "Updating hardware drive mapping...",
        `/map/?start=${start}&end=${end}`);
}

//
function getRumbleLevel() {
    getConfig('/config?item=rumble', function(data) {
        var r = document.getElementById('rumble-level');
        var b = document.getElementById('btRumbleSet');
        var l = data.rumble;
        if (l == null) {
            b.disabled = true;
            r.disabled = true;
            r.value = 0;
        } else {
            b.disabled = false;
            r.disabled = false;
            r.value = l;
        }
        setRumbleHint();
    });
}

//
function setRumbleLevel() {
    var l = document.getElementById('rumble-level').value;
    l = l < 0 ? 0 : l > 255 ? 255 : l;
    putConfig("Please wait", `Setting rumble level to ${l}...`,
        `/config?item=rumble&arg1=${l}`);
}

//
function setRumbleHint() {

    var rl = document.getElementById('rumble-level');
    var rh = document.getElementById('rumble-hint');

    if (rl.disabled) {
        rh.innerHTML = "-";
        return;
    }

    var v = rl.value;
    var h = "off";

    if (v > 200) {
        h = "insane";
    } else if (v > 160) {
        h = "ludicrous";
    } else if (v > 110) {
        h = "ridiculous";
    } else if (v > 70) {
        h = "noisy";
    } else if (v > 45) {
        h = "assertive";
    } else if (v > 30) {
        h = "mellow";
    } else if (v > 20) {
        h = "quiet";
    } else if (v > 0) {
        h = "faint";
    }

    rh.innerHTML = `${v} - ${h}`;
}

//
function getConfig(path, callback) {
    fetch(path, {
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(
        response => response.json()
    ).then(
        data => {
            callback(data);
    }).catch(
        err => console.log('error: ' + err)
    );
}

//
function putConfig(title, message, path) {

    var modInst = userAlert(title, message);

    fetch(path, {
        method: 'PUT'
    }).then(
        response => response.text()
    ).then(
        success => {
            console.log(success);
            modInst.hide();
    }).catch(
        err => console.log('error: ' + err)
    );
}

//
function getVersion() {

    fetch('/version', {
        method: 'GET'
    }).then(
        response => response.text()
    ).then(
        data => {
            document.getElementById('versionLabel').innerHTML = data;
    }).catch(
        err => console.log('error: ' + err)
    );
}
