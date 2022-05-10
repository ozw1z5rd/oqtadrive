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

package base

import (
	"context"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/client"
)

//
type CartridgeImpl interface {
	//
	FS() FileSystem
}

//
type cImpl = CartridgeImpl

//
func NewCartridge(c client.Client, sectorCount int, impl CartridgeImpl) *Cartridge {
	return &Cartridge{
		cImpl:    impl,
		client:   c, // FIXME move
		sectors:  make([]Sector, sectorCount),
		accessIx: sectorCount - 1,
		lock:     make(chan bool, 1),
	}
}

//
type Cartridge struct {
	//
	cImpl
	//
	name           string
	writeProtected bool
	client         client.Client
	//
	sectors   []Sector
	accessIx  int
	modified  bool
	autosaved bool
	//
	lock chan bool
}

//
func (c *Cartridge) Lock(ctx context.Context) bool {
	select {
	case c.lock <- true:
		log.Trace("cartridge locked")
		return true
	case <-ctx.Done():
		log.Debug("cartridge lock timed out")
		return false
	}
}

//
func (c *Cartridge) Unlock() {
	select {
	case <-c.lock:
		log.Trace("cartridge unlocked")
	default:
		log.Debug("cartridge was already unlocked")
	}
}

//
func (c *Cartridge) IsLocked() bool {
	return len(c.lock) > 0
}

//
func (c *Cartridge) Client() client.Client {
	return c.client
}

//
func (c *Cartridge) Name() string {
	return c.name
}

//
func (c *Cartridge) SetName(n string) {
	c.name = n
}

//
func (c *Cartridge) SectorCount() int {
	return len(c.sectors)
}

// SeekToStart sets the access index such that the next call to GetNextSector
// will retrieve the top-most sector, i.e. the sector with the highest sector
// number.
func (c *Cartridge) SeekToStart() {

	if !c.IsFormatted() {
		return
	}

	max := 0
	maxIx := -1

	for ix, sec := range c.sectors {
		if sec != nil && sec.Index() > max {
			max = sec.Index()
			maxIx = ix
		}
	}

	if maxIx > -1 {
		c.accessIx = maxIx
		c.RewindAccessIx(false)
	}
}

//
func (c *Cartridge) Revert() {
	s := c.sectors
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}

// GetNextSector gets the sector at the next access index, skipping slots
// with nil sectors. Access index points to the slot of the returned sector
// afterwards.
func (c *Cartridge) GetNextSector() Sector {
	return c.GetSectorAt(c.AdvanceAccessIx(true))
}

// GetPreviousSector gets the sector at the previous access index, skipping
// slots with nil sectors. Access index points to the slot of the returned
// sector afterwards.
func (c *Cartridge) GetPreviousSector() Sector {
	return c.GetSectorAt(c.RewindAccessIx(true))
}

//
func (c *Cartridge) GetSectorAt(ix int) Sector {
	if 0 <= ix && ix < len(c.sectors) {
		log.Tracef("getting sector at index %d", ix)
		return c.sectors[ix]
	} else {
		log.Tracef("getting sector at invalid index %d", ix)
	}
	return nil
}

// SetNextSector sets the provided sector at the next access index, whether
// there is a sector present at that index or not. Access index points to the
// slot of the set sector afterwards.
func (c *Cartridge) SetNextSector(s Sector) {
	c.SetSectorAt(c.AdvanceAccessIx(false), s)
}

// SetPreviousSector sets the provided sector at the previous access index,
// whether there is a sector present at that index or not. Access index points
// to the slot of the set sector afterwards.
func (c *Cartridge) SetPreviousSector(s Sector) {
	c.SetSectorAt(c.RewindAccessIx(false), s)
}

// setSector sets the provided sector in this cartridge at the given index.
func (c *Cartridge) SetSectorAt(ix int, s Sector) {
	if 0 <= ix && ix < len(c.sectors) {
		log.Tracef("setting sector at index %d", ix)
		c.sectors[ix] = s
		if strings.TrimSpace(s.Name()) != "" {
			c.name = s.Name()
		}
		c.modified = true
	} else {
		log.Errorf("trying to set sector at invalid index %d", ix)
	}
}

//
func (c *Cartridge) IsFormatted() bool {
	for _, s := range c.sectors {
		if s != nil {
			return true
		}
	}
	return false
}

//
func (c *Cartridge) IsWriteProtected() bool {
	return c.writeProtected
}

//
func (c *Cartridge) SetWriteProtected(p bool) {
	c.writeProtected = p
}

//
func (c *Cartridge) IsModified() bool {
	return c.modified
}

//
func (c *Cartridge) SetModified(m bool) {
	c.modified = m
	if m {
		c.autosaved = false
	}
}

//
func (c *Cartridge) IsAutoSaved() bool {
	return c.autosaved
}

//
func (c *Cartridge) SetAutoSaved(a bool) {
	c.autosaved = a
}

//
func (c *Cartridge) AccessIx() int {
	return c.accessIx
}

//
func (c *Cartridge) AdvanceAccessIx(skipEmpty bool) int {
	return c.moveAccessIx(true, skipEmpty)
}

//
func (c *Cartridge) RewindAccessIx(skipEmpty bool) int {
	return c.moveAccessIx(false, skipEmpty)
}

//
func (c *Cartridge) moveAccessIx(forward, skipEmpty bool) int {

	from := c.accessIx

	if !skipEmpty || c.IsFormatted() {
		for {
			if forward {
				c.accessIx = c.ensureIx(c.accessIx - 1)
			} else {
				c.accessIx = c.ensureIx(c.accessIx + 1)
			}
			if !skipEmpty || c.sectors[c.accessIx] != nil {
				break
			}
		}
	}

	log.WithFields(
		log.Fields{"from": from, "to": c.accessIx}).Tracef("moving access ix")

	return c.accessIx
}

//
func (c *Cartridge) ensureIx(ix int) int {
	if ix < 0 {
		return c.SectorCount() - 1 - (-(ix + 1))%c.SectorCount()
	}
	return ix % c.SectorCount()
}

//
func (c *Cartridge) Emit(w io.Writer) {
	c.SeekToStart()
	for ix := 0; ix < c.SectorCount(); ix++ {
		sec := c.GetNextSector()
		if sec != nil {
			sec.Emit(w)
		}
	}
}
