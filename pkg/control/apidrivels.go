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
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/xelalexv/oqtadrive/pkg/daemon"
	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
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

	if a.daemon.GetStatus(drive) == daemon.StatusHardware {
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

	var bytes []byte
	name := getArg(req, "file")
	if info == "dump" && name != "" {
		file, err := cart.FS().Open(name)
		if handleError(err, http.StatusUnprocessableEntity, w) {
			return
		}
		bytes, err = file.Bytes()
		if handleError(err, http.StatusUnprocessableEntity, w) {
			return
		}
	}

	read, write := io.Pipe()

	go func() {
		switch info {

		case "dump":
			if name != "" {
				d := hex.Dumper(write)
				defer d.Close()
				d.Write(bytes)
			} else {
				cart.Emit(write)
			}

		case "ls":
			WriteFileList(write, cart)
		}
		write.Close()
	}()

	sendStreamReply(read, http.StatusOK, w)
}

//
func WriteFileList(w io.Writer, c base.Cartridge) {

	fmt.Fprintf(w, "\n%s\n\n", c.Name())

	if stats, files, err := c.FS().Ls(); err == nil {
		for _, f := range files {
			fmt.Fprintf(w, "%-16s%8d  %-6v\n",
				f.Name(), f.Size(), f.GetAnnotation("file-type"))
		}
		fmt.Fprintf(w, "\n%d of %d sectors used (%dkb free)\n\n",
			stats.Used(), stats.Sectors(), (stats.Sectors()-stats.Used())/2)
	} else {
		fmt.Fprintf(w, "\nerror listing files: %v\n\n", err)
	}
}
