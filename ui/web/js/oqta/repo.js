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
var keyupStack = [];

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

    var button = document.getElementById('btSearch');
    button.onclick = function() {
        search(keyword.value);
    };
}

//
function search(term) {

    if (term.length < 2) {
        return;
    }

    fetch('/search?items=25&term=' + encodeURIComponent(term), {
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(
        response => response.json()
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

    var li = document.createElement('li');
    li.className = "list-group-item text-white bg-dark";
    li.appendChild(document.createTextNode("total hits: " + data.total));
    list.appendChild(li);

    data.hits.forEach(function(l) {
        if (l == "") {
            return;
        }
        li = document.createElement('li');
        li.className = "list-group-item text-white bg-dark";
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

    indicateLoading(drive);
    upload(drive, getName(selectedSearchItem), getFormat(selectedSearchItem),
        selectedSearchItem, true);
    selectedSearchItem = "";

    return true;
}
