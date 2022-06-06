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

package control

import (
	"fmt"
	"net/http"
)

//
func (a *api) getConfig(w http.ResponseWriter, req *http.Request) {

	item := getArg(req, "item")
	conf, err := a.daemon.GetConfig(item)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	if wantsJSON(req) {
		sendJSONReply(map[string]interface{}{item: conf}, http.StatusOK, w)
		return
	}

	sendReply([]byte(fmt.Sprintf("%v", conf)), http.StatusOK, w)
}

//
func (a *api) setConfig(w http.ResponseWriter, req *http.Request) {

	arg1, err := getIntArg(req, "arg1", -1)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	arg2, err := getIntArg(req, "arg2", -1)
	if err != nil {
		arg2 = 0
	}

	if handleError(
		a.daemon.SetConfig(getArg(req, "item"), byte(arg1), byte(arg2)),
		http.StatusUnprocessableEntity, w) {
		return
	}

	sendReply([]byte("configuring"), http.StatusOK, w)
}
