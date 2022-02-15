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

// TODO: JSON response
func (a *api) getDriveMap(w http.ResponseWriter, req *http.Request) {

	start, end, locked := a.daemon.GetHardwareDrives()
	msg := ""

	if start == -1 || end == -1 {
		msg = "no hardware drives"

	} else {
		if start == 0 && end == 0 {
			msg = "hardware drives are off"
		} else {
			msg = fmt.Sprintf("hardware drives: start=%d, end=%d, shadowing=%v",
				start, end, a.daemon.IsShadowingHardwareDrives())
		}
		if locked {
			msg += " (locked)"
		}
	}
	sendReply([]byte(msg), http.StatusOK, w)
}

//
func (a *api) setDriveMap(w http.ResponseWriter, req *http.Request) {

	start, err := getIntArg(req, "start", -1)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	end, err := getIntArg(req, "end", -1)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	shadow := getArg(req, "shadow")

	if shadow == "" { // set drives
		if handleError(a.daemon.MapHardwareDrives(start, end),
			http.StatusUnprocessableEntity, w) {
			return
		}
		sendReply([]byte(fmt.Sprintf(
			"mapped hardware drives: start=%d, end=%d", start, end)),
			http.StatusOK, w)
		return
	}

	// set shadowing
	if start > -1 || end > -1 {
		handleError(fmt.Errorf("don't set shadowing while setting drives"),
			http.StatusNotAcceptable, w)
		return
	}

	if handleError(a.daemon.ShadowHardwareDrives(shadow == "true"),
		http.StatusUnprocessableEntity, w) {
		return
	}

	if shadow == "true" {
		shadow = "on"
	} else {
		shadow = "off"
	}

	sendReply([]byte(fmt.Sprintf(
		"switched hardware drive shadowing %s", shadow)), http.StatusOK, w)
	return
}
