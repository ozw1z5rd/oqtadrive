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
	"io"
	"net/http"
	"strings"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/format"
	"github.com/xelalexv/oqtadrive/pkg/repo"
	"github.com/xelalexv/oqtadrive/pkg/util"
)

//
func (a *api) load(w http.ResponseWriter, req *http.Request) {

	drive := getDrive(w, req)
	if drive == -1 {
		return
	}

	var in io.ReadCloser

	if ref, err := getRef(req); ref != "" {
		if err == nil {
			in, err = repo.Resolve(ref, a.repository)
		}
		if err != nil {
			handleError(err, http.StatusNotAcceptable, w)
			return
		}
	} else {
		in = http.MaxBytesReader(nil, req.Body, 1048576) // FIXME make constant
	}

	cr, err := format.NewCartReader(in, getArg(req, "compressor"))
	if err != nil {
		handleError(err, http.StatusUnprocessableEntity, w)
		return
	}
	defer cr.Close()

	typ := getArg(req, "type")
	if typ == "" {
		typ = cr.Type()
	}

	reader, err := format.NewFormat(typ)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	params := util.Params{
		"name":     getArg(req, "name"),
		"launcher": getArg(req, "launcher"),
	}
	cart, err := reader.Read(cr, true, isFlagSet(req, "repair"), params)
	if err != nil {
		handleError(fmt.Errorf("cartridge corrupted: %v", err),
			http.StatusUnprocessableEntity, w)
		return
	}

	if handleError(req.Body.Close(), http.StatusInternalServerError, w) {
		return
	}

	if err := a.daemon.SetCartridge(drive, cart, isFlagSet(req, "force")); err != nil {
		if strings.Contains(err.Error(), "could not lock") {
			handleError(fmt.Errorf("drive %d busy", drive), http.StatusLocked, w)
		} else if strings.Contains(err.Error(), "is modified") {
			handleError(fmt.Errorf(
				"cartridge in drive %d is modified", drive), http.StatusConflict, w)
		} else {
			handleError(err, http.StatusInternalServerError, w)
		}

	} else {
		sendReply([]byte(
			fmt.Sprintf("loaded data into drive %d", drive)), http.StatusOK, w)
		a.forceNotify <- true
	}
}
