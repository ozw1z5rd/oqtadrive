/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2021, Alexander Vollschwitz

   This file is part of OqtaDrive.

   The Z80toMDR code is based on Z80onMDR_Lite, copyright (c) 2021 Tom Dalby,
   ported from C to Go by Alexander Vollschwitz. For the original C code, refer
   to:

        https://github.com/TomDDG/Z80onMDR_lite

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

package z80

import (
	"bytes"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
	"github.com/xelalexv/oqtadrive/pkg/microdrive/if1"
	"github.com/xelalexv/oqtadrive/pkg/microdrive/raw"
)

//
func (s *snapshot) pack() error {

	s.cart = if1.NewCartridge()
	s.cart.SetName(s.name)
	var wg sync.WaitGroup

	// screen
	scr := &part{}
	wg.Add(1)

	packScreen := func(main []byte) {
		scr.data = make([]byte, 6912+216+109)
		scr.length = zxsc(main, scr.data[len(scrLoad):], 6912, true, "0")
		scr.length += copy(scr.data, scrLoad) // add m/c
		scr.file = fmt.Sprintf("%-10s", "0")
		scr.start = 32179
		scr.param = 0xffff
		scr.dataType = 0x03
		log.WithField("size", scr.length).Debug("screen file")
	}

	go func() {
		defer wg.Done()
		packScreen(s.main)
	}()

	// otek pages
	var pg1 *part
	var pg3 *part
	var pg4 *part
	var pg6 *part
	var pg7 *part

	if s.launcher.isOtek() {

		lenData := 16384 + 512 + len(unpack)

		// page 1
		pg1 = &part{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			pg1.data = make([]byte, lenData)
			pg1.length = zxsc(
				s.main[s.bank[4]:], pg1.data[len(unpack):], 16384, false, "1")
			copy(pg1.data, unpack) // add in unpacker
			pg1.length += len(unpack)
			pg1.start = 32256 - len(unpack)
			pg1.param = 0xffff
			pg1.file = fmt.Sprintf("%-10d", 1)
			pg1.dataType = 0x03
			log.WithField("size", pg1.length).Debug("page file 1")
		}()

		// page 3
		pg3 = &part{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			pg3.data = make([]byte, lenData)
			pg3.data[0] = 0x13
			pg3.length = zxsc(s.main[s.bank[6]:], pg3.data[1:], 16384, false, "2") + 1
			pg3.start = 32255 // don't need to replace the unpacker, just the page number
			pg3.param = 0xffff
			pg3.file = fmt.Sprintf("%-10d", 2)
			pg3.dataType = 0x03
			log.WithField("size", pg3.length).Debug("page file 3")
		}()

		// page 4
		pg4 = &part{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			pg4.data = make([]byte, lenData)
			pg4.data[0] = 0x14
			pg4.length = zxsc(s.main[s.bank[7]:], pg4.data[1:], 16384, false, "3") + 1
			pg4.file = fmt.Sprintf("%-10d", 3)
			pg4.start = 32255
			pg4.param = 0xffff
			pg4.dataType = 0x03
			log.WithField("size", pg4.length).Debug("page file 4")
		}()

		// page 6
		pg6 = &part{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			pg6.data = make([]byte, lenData)
			pg6.data[0] = 0x16
			pg6.length = zxsc(s.main[s.bank[9]:], pg6.data[1:], 16384, false, "4") + 1
			pg6.file = fmt.Sprintf("%-10d", 4)
			pg6.start = 32255
			pg6.param = 0xffff
			pg6.dataType = 0x03
			log.WithField("size", pg6.length).Debug("page file 6")
		}()

		// page 7
		pg7 = &part{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			pg7.data = make([]byte, lenData)
			pg7.data[0] = 0x17
			pg7.length = zxsc(s.main[s.bank[10]:], pg7.data[1:], 16384, false, "5") + 1
			pg7.file = fmt.Sprintf("%-10d", 5)
			pg7.start = 32255
			pg7.param = 0xffff
			pg7.dataType = 0x03
			log.WithField("size", pg7.length).Debug("page file 7")
		}()
	}

	// runner & main
	main := &part{}
	run := &part{}
	wg.Add(1)

	// The hidden launcher modifies main, so we need to give it its own copy to
	// ensure we don't disturb the packing of the other parts above, which can
	// happen in parallel on multi-core systems.
	mainCp := make([]byte, len(s.main))
	copy(mainCp, s.main)

	go func() {
		defer wg.Done()
		main.data = make([]byte, s.launcher.mainSize()+10240)

		delta := 3
		dgap := 0

		for {
			main.err = s.launcher.byteSeriesScan(mainCp, delta, dgap)
			if main.err != nil {
				return
			}
			// up to the full size - delta
			main.length = zxsc(mainCp[s.launcher.startPos():], main.data[8704:],
				s.launcher.mainSize()-delta, false, "M")
			dgap = decompressf(main.data[8704:], main.length, s.launcher.mainSize())
			delta += dgap
			if delta > BGap {
				main.err = fmt.Errorf(
					"cannot compress main block, delta too large: %d > %d",
					delta, BGap)
				return
			}
			if dgap < 1 {
				break
			}
		}

		maxSize := 40624 // 0x6150 onwards
		var adder int
		adder, main.err = s.launcher.getAdder(delta, main.length, maxSize)
		if main.err != nil {
			return
		}
		main.length += adder

		s.launcher.flushMain(main.data, mainCp, adder, delta)
		main.data = main.data[8704-adder:]
		main.start = 65536 - main.length
		main.param = 0xffff
		main.file = fmt.Sprintf("%-10s", "M")
		main.dataType = 0x03
		log.WithFields(log.Fields{
			"size": main.length, "delta": delta}).Debug("main file")

		// run file
		s.launcher.flushRun(run)
		run.file = fmt.Sprintf("%-10s", "run")
		run.start = 23813
		run.param = 0
		run.dataType = 0x00
		log.WithField("size", run.length).Debug("run file")
	}()

	wg.Wait()

	// if stack is within screen, we need to repack the screen or we're
	// missing the launcher
	if 0 < s.launcher.stackPos() && s.launcher.stackPos() <= 23296 {
		scr = &part{}
		packScreen(mainCp)
	}

	parts := []*part{run, scr, pg1, pg3, pg4, pg6, pg7, main}

	// position access index at top most sector
	s.cart.SeekToStart()
	s.cart.AdvanceAccessIx(false)

	for _, p := range parts {
		if err := addToCartridge(s.cart, p); err != nil {
			return err
		}
	}

	if err := fillBlanks(s.cart); err != nil {
		return err
	}

	return nil
}

// TODO: currently not used, maybe remove?
func interleave(parts []*part) []*part {

	count := 0
	for _, p := range parts {
		if p != nil {
			count++
		}
	}

	ret := make([]*part, count)
	ix := 0

	for _, p := range parts {
		if p != nil {
			ret[ix] = p
			ix += 2
			if ix >= len(ret) {
				ix = 1
			}
		}
	}

	return ret
}

//
type part struct {
	file     string
	data     []byte
	length   int
	start    int
	param    int
	dataType byte
	err      error
}

// add data to the virtual cartridge
func addToCartridge(cart base.Cartridge, p *part) error {

	if p == nil {
		return nil
	}

	if p.err != nil {
		return p.err
	}

	log.WithFields(log.Fields{
		"file":   p.file,
		"length": p.length,
		"start":  p.start,
		"param":  p.param,
		"type":   p.dataType,
	}).Debug("adding to cartridge")

	var dataPos int
	var sPos int

	// work out how many sectors needed
	numSec := ((p.length + 9) / 512) + 1 // +9 for initial header

	for sequence := 0; sequence < numSec; sequence++ {

		var b bytes.Buffer

		// sector header
		raw.WriteSyncPattern(&b)
		b.WriteByte(0x01)
		secIx := cart.AccessIx()
		b.WriteByte(byte(secIx + 1))
		b.WriteByte(0x00)
		b.WriteByte(0x00)
		b.WriteString(cart.Name())
		b.WriteByte(0x00)

		hd, _ := if1.NewHeader(b.Bytes(), false)
		if err := hd.FixChecksum(); err != nil {
			return fmt.Errorf("error creating header: %v", err)
		}

		// file header
		//	0x06 - for end of file and data, 0x04 for data if in numerous parts
		//	0x00 - sequence number (if file in many parts then this is the number)
		//	0x00 0x00 - length of this part 16bit
		//	0x00*10 - filename
		//	0x00 - header checksum
		b.Reset()
		raw.WriteSyncPattern(&b)
		if sequence == numSec-1 {
			b.WriteByte(0x06)
		} else {
			b.WriteByte(0x04)
		}
		b.WriteByte(byte(sequence))

		num := 0
		if p.length > 512 { // if length >512 then this is 512 until final part
			num = 512
		} else if numSec > 1 {
			num = p.length
		} else {
			num = p.length + 9 // add 9 for header info
		}
		writeUInt16(&b, num)

		b.WriteString(p.file)
		b.WriteByte(0x00)

		// data - 512 bytes of data
		//
		// *note first sequence of data must have the header in the format
		//
		//  (1)   0x00, 0x01, 0x02 or 0x03 - program, number array, character
		//        array or code file
		//  (2,3) 0x00 0x00 - total length
		//  (4,5) start address of the block (0x05 0x5d for basic 23813)
		//  (6,7) 0x00 0x00 - total length of program (same as above if
		//        basic of 0xff if code)
		//  (8,9) 0x00 0x00 - line number if LINE used
		//
		if sequence == 0 {
			b.WriteByte(p.dataType)
			writeUInt16(&b, p.length)
			writeUInt16(&b, p.start)

			if p.dataType == 0x00 { // basic
				writeUInt16(&b, p.length)
				writeUInt16(&b, p.param)
			} else {
				b.WriteByte(0xff)
				b.WriteByte(0xff)
				b.WriteByte(0xff)
				b.WriteByte(0xff)
			}

			sPos = 36

		} else {
			sPos = 27 // to cover the headers
		}

		j := p.length // copy code

		if j > 512 {
			j = 512
			if sequence == 0 {
				j -= 9
			}
		}

		for i := 0; i < j; i++ {
			b.WriteByte(p.data[dataPos])
			dataPos++
			sPos++
		}

		for ; sPos < if1.RecordLength; sPos++ { // padding on last sequence
			b.WriteByte(0x00)
		}

		if sequence == 0 {
			p.length -= 503
		} else {
			p.length -= 512
		}

		rec, _ := if1.NewRecord(b.Bytes(), false)
		if err := rec.FixChecksums(); err != nil {
			return fmt.Errorf("error creating record: %v", err)
		}

		if sec, err := base.NewSector(hd, rec); err != nil {
			return err
		} else {
			cart.SetSectorAt(secIx, sec)
			if err := advanceWithInterleave(cart); err != nil {
				return err
			}
		}
	}

	// additional sector gap after each file
	if err := advanceWithInterleave(cart); err != nil {
		return err
	}

	return nil
}

//
func advanceWithInterleave(cart base.Cartridge) error {

	cart.AdvanceAccessIx(false)
	ix := cart.AdvanceAccessIx(false) // sector interleave

	if cart.GetSectorAt(ix) != nil {
		// When seeing a non-nil sector, we've made one round through the
		// cartridge and there's an even number of sectors, i.e. we've hit the
		// first sector we added. In this case, advance access index by one.
		// For an odd number of sectors, we will be aligned with the nil sectors
		// automatically in the second round.
		ix = cart.AdvanceAccessIx(false)
		if cart.GetSectorAt(ix) != nil {
			// if still non-nil, cartridge is full
			return fmt.Errorf("cartridge full")
		}
	}

	return nil
}

//
func fillBlanks(cart base.Cartridge) error {

	for ix := 0; ix < cart.SectorCount(); ix++ {
		if cart.GetSectorAt(ix) == nil {
			if err := addBlankSectorAt(cart, ix); err != nil {
				return err
			}
		}
	}

	return nil
}

//
func addBlankSectorAt(cart base.Cartridge, ix int) error {

	var b bytes.Buffer

	// sector header
	raw.WriteSyncPattern(&b)
	b.WriteByte(0x01)
	b.WriteByte(byte(ix + 1))
	b.WriteByte(0x00)
	b.WriteByte(0x00)
	b.WriteString(cart.Name())
	b.WriteByte(0x00)

	hd, _ := if1.NewHeader(b.Bytes(), false)
	if err := hd.FixChecksum(); err != nil {
		return err
	}

	// blank record
	rec, _ := if1.NewRecord(make([]byte, if1.RecordLength), false)
	if err := rec.FixChecksums(); err != nil {
		return err
	}

	if sec, err := base.NewSector(hd, rec); err != nil {
		return err
	} else {
		cart.SetSectorAt(ix, sec)
	}

	return nil
}
