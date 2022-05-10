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

package if1

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
	"github.com/xelalexv/oqtadrive/pkg/util"
)

//
type file struct {
	*base.File
}

//
func (f *file) FileHeaderLength() int {
	return FileHeaderLength
}

//
func newFs(cart *base.Cartridge) *fsys {
	return &fsys{cart: cart}
}

//
type fsys struct {
	cart *base.Cartridge
}

//
func (fs *fsys) Open(name string) (*base.File, error) {

	var records []base.Record
	size := 0

	for ix := 0; ix < fs.cart.SectorCount(); ix++ {

		if s := fs.cart.GetSectorAt(ix); s != nil {
			if r := s.Record(); r != nil {

				if r.Flags()&RecordFlagsUsed == 0 {
					continue
				}

				n := strings.TrimSpace(translate(r.Name()))
				if n != name {
					continue
				}

				if d := r.Index() + 1 - len(records); d > 0 {
					records = append(records, make([]base.Record, d)...)
				}
				records[r.Index()] = r
				size += r.Length()
			}
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("file not found")
	}

	impl := &file{}
	ret := base.NewFile(name, size, records, impl)
	impl.File = ret

	return ret, nil
}

//
func (fs *fsys) Ls() (*base.FsStats, []*base.FileInfo, error) {

	dir := make(map[string]int)
	anno := make(map[string]util.Annotations)
	used := 0

	for ix := 0; ix < fs.cart.SectorCount(); ix++ {

		if sec := fs.cart.GetNextSector(); sec != nil {
			if rec := sec.Record(); rec != nil {

				if rec.Flags()&RecordFlagsUsed == 0 {
					continue
				}

				used++

				name := translate(rec.Name())
				if name == "" {
					continue
				}

				size, ok := dir[name]
				if ok {
					size += rec.Length()
				} else {
					size = rec.Length()
				}
				dir[name] = size
				if rec.Index() == 0 {
					anno[name] = fileAnnotations(rec.(*record))
				}
			}
		}
	}

	var files []string
	for f := range dir {
		files = append(files, f)
	}
	sort.Strings(files)

	ret := make([]*base.FileInfo, len(files))
	for ix, name := range files {
		ret[ix] = base.NewFileInfo(name, dir[name])
		ret[ix].Annotations = anno[name]
	}

	return base.NewFsStats(fs.cart.SectorCount(), used), ret, nil
}

//
func fileAnnotations(r *record) util.Annotations {

	a := make(util.Annotations)

	t := "?"
	switch r.block.GetByte("fileType") {
	case 0:
		t = "BASIC"
	case 1:
		fallthrough
	case 2:
		t = "array"
	case 3:
		t = "code"
	}
	a.Annotate("file-type", t)
	a.Annotate("line", r.block.GetInt("lineNumber"))

	return a
}
