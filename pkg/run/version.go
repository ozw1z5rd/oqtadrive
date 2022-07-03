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

package run

import (
	"fmt"
	"io"
	"strings"

	"github.com/xelalexv/oqtadrive/pkg/util"
)

//
func NewVersion() *Version {
	v := &Version{}
	v.Runner = *NewRunner(
		"version", "get daemon & adapter version info", "", "", "", v.Run)
	v.AddBaseSettings()
	return v
}

//
type Version struct {
	Runner
}

//
func (v *Version) Run() error {

	resp, err := v.apiCall("GET", "/version", false, nil)
	if err != nil {
		PrintVersion("daemon:     not reachable\n")
		return nil
	}
	defer resp.Close()

	buf := new(strings.Builder)
	if _, err = io.Copy(buf, resp); err != nil {
		return err
	}

	PrintVersion(buf.String())
	return nil
}

//
func PrintVersion(remote string) {
	fmt.Printf(`
   ___        _        ____       _
  / _ \  __ _| |_ __ _|  _ \ _ __(_)_   _____
 | | | |/ _' | __/ _' | | | | '__| \ \ / / _ \
 | |_| | (_| | || (_| | |_| | |  | |\ V /  __/
  \___/ \__, |\__\__,_|____/|_|  |_| \_/ \___|
           |_|

 dedicated to Sir Clive Sinclair (30 July 1940 - 16 September 2021)

oqtactl:    %s
`, util.OqtaDriveVersion)
	if remote != "" {
		fmt.Printf("%s", remote)
	}
	fmt.Println()
}
