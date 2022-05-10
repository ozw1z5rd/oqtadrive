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

package run

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/format"
)

//
func NewDump() *Dump {

	d := &Dump{}
	d.Runner = *NewRunner(
		"dump [-d|--drive {drive}] [-f|--file {file}] [-i|--input {file}] [-a|--address {address}]",
		"dump cartridge from file or daemon",
		"\nUse the dump command to output a hex dump for a cartridge from file or from daemon.",
		"", runnerHelpEpilogue, d.Run)

	d.AddBaseSettings()
	d.AddSetting(&d.Input, "input", "i", "", nil, "cartridge input file", false)
	d.AddSetting(&d.Drive, "drive", "d", "", 1, "drive number (1-8)", false)
	d.AddSetting(&d.File, "file", "f", "", nil, "file on cartridge to dump", false)

	return d
}

//
type Dump struct {
	//
	Runner
	//
	Drive int
	Input string
	File  string
}

//
func (d *Dump) Run() error {

	d.ParseSettings()

	if d.Input != "" {
		f, err := os.Open(d.Input)
		if err != nil {
			return err
		}
		defer f.Close()

		_, typ, comp := format.SplitNameTypeCompressor(d.Input)

		rd, err := format.NewCartReader(ioutil.NopCloser(bufio.NewReader(f)), comp)
		if err != nil {
			return err
		}

		if typ == "" {
			typ = rd.Type()
		}

		form, err := format.NewFormat(typ)
		if err != nil {
			return err
		}

		cart, err := form.Read(rd, false, false, nil)
		if err != nil {
			return err
		}

		if d.File != "" {
			f, err := cart.FS().Open(d.File)
			if err != nil {
				return err
			}
			if bytes, err := f.Bytes(); err == nil {
				d := hex.Dumper(os.Stdout)
				defer d.Close()
				d.Write(bytes)
			} else {
				return err
			}
		} else {
			cart.Emit(os.Stdout)
		}

	} else {
		if err := validateDrive(d.Drive); err != nil {
			return err
		}

		resp, err := d.apiCall("GET",
			fmt.Sprintf("/drive/%d/dump?file=%s", d.Drive, d.File), false, nil)
		if err != nil {
			return err
		}
		defer resp.Close()

		if d.File != "" {
			if bytes, err := ioutil.ReadAll(resp); err == nil {
				d := hex.Dumper(os.Stdout)
				defer d.Close()
				d.Write(bytes)
			} else {
				return err
			}
		} else if _, err := io.Copy(os.Stdout, resp); err != nil {
			return err
		}
	}

	fmt.Println()
	return nil
}
