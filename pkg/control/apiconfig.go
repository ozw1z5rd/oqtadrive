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
	"strings"

	"github.com/xelalexv/oqtadrive/pkg/daemon"
)

//
func (a *api) getConfig(w http.ResponseWriter, req *http.Request) {

	var items []string

	if i := getArg(req, "item"); i != "" {
		items = append(items, i)
	} else {
		// add all currently used config items here
		items = append(items, daemon.CmdConfigItemRumble)
	}

	configs := make(map[string]interface{})

	for _, i := range items {
		conf, err := a.daemon.GetConfig(i)
		if handleError(err, http.StatusUnprocessableEntity, w) {
			return
		}
		configs[i] = conf
	}

	if wantsJSON(req) {
		sendJSONReply(configs, http.StatusOK, w)
		return
	}

	var buf strings.Builder
	for k, v := range configs {
		fmt.Fprintf(&buf, "%s = %v", k, v)
	}
	sendReply([]byte(buf.String()), http.StatusOK, w)
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
