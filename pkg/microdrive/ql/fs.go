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

package ql

import (
	"fmt"
	"sort"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
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
func newFs(cart *cartridge) *fsys {
	return &fsys{cart: cart}
}

//
type fsys struct {
	cart *cartridge
}

//
func (fs *fsys) Open(name string) (*base.File, error) {

	index := make(map[int]int)
	var zero base.Sector
	var first *record

	for ix := 0; ix < fs.cart.SectorCount(); ix++ {

		if s := fs.cart.GetSectorAt(ix); s != nil {
			index[s.Index()] = ix
			if r := s.Record().(*record); r != nil {
				// documentation states that sector map is located in sector 0,
				// bearing file number 0xf8, but 0x80 is also often observed,
				// not sure why; we need to consider both
				if s.Index() == 0 && (r.Flags() == 0xF8 || r.Flags() == 0x80) {
					if zero != nil {
						return nil, fmt.Errorf("more than one zero block found")
					}
					zero = s
					continue
				}
				if r.Flags() > 0xf0 || r.Index() > 0 || r.Name() != name {
					continue
				}
				if first != nil {
					return nil, fmt.Errorf("more than one first block found")
				}
				first = r
			}
		}
	}

	if zero == nil {
		return nil, fmt.Errorf("zero block not found")
	}
	if first == nil {
		return nil, fmt.Errorf("file not found")
	}

	sm, err := newSectorMap(fs.cart, zero, index)
	if err != nil {
		return nil, err
	}

	records, err := sm.collectFileRecords(int(first.Flags()))
	if err != nil {
		return nil, err
	}

	impl := &file{}
	ret := base.NewFile(name, first.Length(), records, impl)
	impl.File = ret

	return ret, nil
}

//
func (fs *fsys) Ls() (*base.FsStats, []*base.FileInfo, error) {

	dir := make(map[string]int)
	used := fs.cart.SectorCount()

	for ix := 0; ix < fs.cart.SectorCount(); ix++ {
		if sec := fs.cart.GetNextSector(); sec != nil {
			if rec := sec.Record(); rec != nil {
				if rec.Flags() == 0xfd {
					used--
				}
				if rec.Flags() > 0xf0 || rec.Index() > 0 {
					continue
				}
				dir[rec.Name()] = rec.Length() // FIXME: do things like this belong into FS methods?
			}
		}
	}

	var files []string
	for f := range dir {
		if f != "" {
			files = append(files, f)
		}
	}
	sort.Strings(files)

	ret := make([]*base.FileInfo, len(files))
	for ix, name := range files {
		ret[ix] = base.NewFileInfo(name, dir[name])
	}

	return base.NewFsStats(fs.cart.SectorCount(), used), ret, nil
}

//
func newSectorMap(cart base.Cartridge, s base.Sector,
	index map[int]int) (*sectorMap, error) {

	if s.Record() == nil {
		return nil, fmt.Errorf("empty sector map sector")
	}
	sectors := s.Record().(*record).Data()
	if sectors[0] != 0xf8 && sectors[1] != 0x00 {
		return nil, fmt.Errorf("not a sector map")
	}
	return &sectorMap{cart: cart, sectors: sectors, index: index}, nil
}

//
type sectorMap struct {
	cart    base.Cartridge
	sectors []byte
	index   map[int]int
}

//
func (sm *sectorMap) getSector(ix int) (int, int) {
	if 0 <= ix && 2*ix < len(sm.sectors) {
		return int(sm.sectors[ix*2]), int(sm.sectors[ix*2+1])
	}
	return -1, -1
}

//
func (sm *sectorMap) collectFileRecords(number int) ([]base.Record, error) {

	var records []base.Record

	for s := 0; s < SectorCount; s++ {

		if fNum, rNum := sm.getSector(s); fNum == number {
			if d := rNum + 1 - len(records); d > 0 {
				records = append(records, make([]base.Record, d)...)
			}
			if ix, ok := sm.index[s]; ok {
				if r := sm.cart.GetSectorAt(ix).Record().(*record); r != nil {
					records[rNum] = r
				} else {
					return nil, fmt.Errorf("no record in sector: %d", s)
				}
			} else {
				return nil, fmt.Errorf("sector not found: %d", s)
			}
		}
	}

	return records, nil
}
