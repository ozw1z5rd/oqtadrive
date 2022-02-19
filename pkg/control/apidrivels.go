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

	"github.com/xelalexv/oqtadrive/pkg/daemon"
)

//
func (a *api) dump(w http.ResponseWriter, req *http.Request) {
	a.driveInfo(w, req, "dump")
}

//
func (a *api) driveList(w http.ResponseWriter, req *http.Request) {
	a.driveInfo(w, req, "ls")
}

//
func (a *api) driveInfo(w http.ResponseWriter, req *http.Request, info string) {

	drive := getDrive(w, req)
	if drive == -1 {
		return
	}

	hw := a.daemon.GetStatus(drive) == daemon.StatusHardware
	if hw && !a.daemon.IsShadowingHardwareDrives() {
		sendReply([]byte(fmt.Sprintf(
			"hardware drive mapped to slot %d", drive)),
			http.StatusOK, w)
		return
	}

	cart, ok := a.daemon.GetCartridge(drive)

	if !ok {
		handleError(fmt.Errorf("drive %d busy", drive), http.StatusLocked, w)
		return
	}

	if cart == nil {
		handleError(fmt.Errorf("no cartridge in drive %d", drive),
			http.StatusUnprocessableEntity, w)
		return
	}

	defer cart.Unlock()

	read, write := io.Pipe()

	go func() {

		switch info {

		case "dump":
			cart.Emit(write)

		case "ls":
			if hw {
				fmt.Fprintf(write, "\nshadowed hardware drive\n")
			}

			cart.List(write)

			if cart.HasAnnotation("health.sectors") {
				health := 0
				sec := cart.GetAnnotation("health.sectors").Int()
				if sec > 0 {
					health = 100 * (sec - cart.GetAnnotation("health.sectors.bad").Int()) / sec
				}
				fmt.Fprintf(write,
					"sector health: %d%%\n  %d bad sectors (%d headers, %d records)\n\n",
					health,
					cart.GetAnnotation("health.sectors.bad").Int(),
					cart.GetAnnotation("health.headers.bad").Int(),
					cart.GetAnnotation("health.records.bad").Int())
			}

		}
		write.Close()
	}()

	sendStreamReply(read, http.StatusOK, w)
}
