/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2022, Alexander Vollschwitz

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
	"bufio"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

//
func newLauncher(typ string) (launcher, error) {

	var l launcher

	switch typ {
	case "screen":
		l = &lScreen{}
	case "hidden":
		fallthrough
	case "":
		typ = "hidden"
		l = &lHidden{}
	default:
		return nil, fmt.Errorf("unsupported launcher type: '%s'", typ)
	}

	log.WithField("launcher", typ).Debugf("launcher created")
	return l, nil
}

//
type launcher interface {
	setup(rd *bufio.Reader, sna bool, size int) error
	postSetup(rd *bufio.Reader) (byte, error)
	byteSeriesScan(main []byte, delta, dgap int) error
	getAdder(delta, cmSize, maxSize int) (int, error)
	flushRun(r *part)
	flushMain(comp, main []byte, adder, delta int)
	isCompressed() bool
	hardwareMode() byte
	isOtek() bool
	addLength() int
	startPos() int
	// stackPos is the absolute stack pointer int the 64k address space;
	// it corresponds to `noc_launchstk_pos` in the C version, and is
	// `len(nocLaunchStk)` lower than `stackpos` from the C version
	stackPos() int
	adjustStackPos(main []byte, sna bool) (bool, error)
	mainSize() int
}

// --- screen based launcher --------------------------------------------------
// usually exhibits some amount of screen corruption at top of screen
//
type lScreen struct {
	//
	bln []byte
	scr []byte
	prt []byte
	igp []byte
	stk []byte
	//
	igpPos int
	vgap   byte
	//
	compressed bool
	otek       bool
	addLen     int
	stkPos     int
	hwMode     byte
}

//
func (s *lScreen) isCompressed() bool {
	return s.compressed
}

//
func (s *lScreen) hardwareMode() byte {
	return s.hwMode
}

//
func (s *lScreen) isOtek() bool {
	return s.otek
}

//
func (s *lScreen) addLength() int {
	return s.addLen
}

//
func (s *lScreen) startPos() int {
	return 6966 - len(nocLaunchPrt)
}

//
func (s *lScreen) stackPos() int {
	return s.stkPos
}

//
func (s *lScreen) mainSize() int {
	return 42186 + len(nocLaunchPrt)
}

//
func (s *lScreen) setup(rd *bufio.Reader, sna bool, size int) error {

	s.bln = make([]byte, len(mdrBln))
	copy(s.bln, mdrBln)
	s.scr = make([]byte, len(launchScr))
	copy(s.scr, launchScr)
	s.prt = make([]byte, len(nocLaunchPrt))
	copy(s.prt, nocLaunchPrt)
	s.igp = make([]byte, len(nocLaunchIgp))
	copy(s.igp, nocLaunchIgp)
	s.stk = make([]byte, len(nocLaunchStk))
	copy(s.stk, nocLaunchStk)

	if sna {
		return s.setupSNA(rd, size)
	} else {
		return s.setupZ80(rd)
	}
}

//
func (s *lScreen) setupZ80(rd *bufio.Reader) error {

	var c byte
	var err error

	if err = fill(rd, s.scr, []int{ // read in z80 starting with header
		launchScrAF + 1, //	0   1    A register
		launchScrAF,     //	1   1    F register
		launchScrBC,     //	2   2    BC register pair(LSB, i.e.C, first)
		launchScrBC + 1,
		launchScrHL, //		4   2    HL register pair
		launchScrHL + 1,
		launchScrJP, //		6   2    Program counter (if zero then version 2 or 3 snapshot)
		launchScrJP + 1,
		launchScrSP, //		8   2    Stack pointer
		launchScrSP + 1,
	}); err != nil {
		return err
	}

	s.stk[nocLaunchStkAF+1] = s.scr[launchScrAF+1]
	s.stk[nocLaunchStkAF] = s.scr[launchScrAF]
	s.stk[nocLaunchStkBC] = s.scr[launchScrBC]
	s.stk[nocLaunchStkBC+1] = s.scr[launchScrBC+1]
	s.stk[nocLaunchStkHL] = s.scr[launchScrHL]
	s.stk[nocLaunchStkHL+1] = s.scr[launchScrHL+1]
	s.stk[nocLaunchStkJP] = s.scr[launchScrJP]
	s.stk[nocLaunchStkJP+1] = s.scr[launchScrJP+1]

	if s.stkPos = int(s.scr[launchScrSP+1])*256 + int(s.scr[launchScrSP]); s.stkPos == 0 {
		s.stkPos = 65536
	}
	s.stkPos -= len(nocLaunchStk) // pos of stack code

	p := s.stkPos + nocLaunchStkAF
	s.scr[launchScrSP] = byte(p)
	s.scr[launchScrSP+1] = byte(p >> 8) // start of stack within stack
	s.igp[nocLaunchIgpRD] = byte(p)
	s.igp[nocLaunchIgpRD+1] = byte(p >> 8)

	// 10   1    Interrupt register
	if s.bln[mdrBlnI], err = rd.ReadByte(); err != nil {
		return err
	}

	// 11   1    Refresh register (Bit 7 is not significant!)
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	s.scr[launchScrR] = c - 4    // r, reduce by 4 so correct on launch
	s.stk[nocLaunchStkR] = c - 3 // 3 for 4 stage launcher

	//  12   1    Bit 0: Bit 7 of r register; Bit 1-3: Border colour;
	//            Bit 4=1: SamROM; Bit 5=1:v1 Compressed; Bit 6-7: N/A
	if c, err = rd.ReadByte(); err != nil {
		return err
	}

	s.compressed = (c&32)>>5 == 1 // 1 compressed, 0 not

	if c&1 == 1 || c > 127 {
		s.scr[launchScrR] |= 128 // r high bit set
		s.stk[nocLaunchStkR] |= 128
	} else {
		s.scr[launchScrR] &= 127 // r high bit reset
		s.stk[nocLaunchStkR] &= 127
	}

	s.bln[mdrBlnBRD] = ((c & 14) >> 1) + 0x30                   // border
	s.bln[mdrBlnPAP] = (((c & 14) >> 1) << 3) + ((c & 14) >> 1) // paper/ink

	if err = fill(rd, s.scr, []int{
		launchScrDE, //      13   2    DE register pair
		launchScrDE + 1,
	}); err != nil {
		return err
	}

	s.igp[nocLaunchIgpDE] = s.scr[launchScrDE]
	s.igp[nocLaunchIgpDE+1] = s.scr[launchScrDE+1]

	if err = fill(rd, s.bln, []int{
		mdrBlnBCA, //     15   2    BC' register pair
		mdrBlnBCA + 1,
		mdrBlnDEA, //     17   2    DE' register pair
		mdrBlnDEA + 1,
		mdrBlnHLA, //     19   2    HL' register pair
		mdrBlnHLA + 1,
		mdrBlnAFA + 1, // 21   1    A' register
		mdrBlnAFA,     // 22   1    F' register
		mdrBlnIY,      // 23   2    IY register (Again LSB first)
		mdrBlnIY + 1,
		mdrBlnIX, //      25   2    IX register
		mdrBlnIX + 1,
	}); err != nil {
		return err
	}

	// 27   1    Interrupt flip flop, 0 = DI, otherwise EI
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	if c == 0 {
		s.scr[launchScrEI] = 0xf3 // di
		s.stk[nocLaunchStkEI] = 0xf3
	} else {
		s.scr[launchScrEI] = 0xfb // ei
		s.stk[nocLaunchStkEI] = 0xfb
	}

	// 28   1    IFF2 [IGNORED]
	if _, err = rd.ReadByte(); err != nil {
		return err
	}

	// 29   1    Bit 0-1: IM(0, 1 or 2); Bit 2-7: N/A
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	c &= 3
	if c == 0 {
		s.bln[mdrBlnIM] = 0x46 // im 0
	} else if c == 1 {
		s.bln[mdrBlnIM] = 0x56 // im 1
	} else {
		s.bln[mdrBlnIM] = 0x5e // im 2
	}

	// version 2 & 3 only
	s.addLen = 0 // 0 indicates v1, 23 for v2 otherwise v3
	s.otek = false

	if s.scr[launchScrJP] == 0 && s.scr[launchScrJP+1] == 0 {

		// 30   2    Length of additional header block
		if s.addLen, err = readUInt16(rd); err != nil {
			return err
		}

		// 32   2    Program counter
		if err = fill(rd, s.scr, []int{
			launchScrJP,
			launchScrJP + 1,
		}); err != nil {
			return err
		}
		s.stk[nocLaunchStkJP] = s.scr[launchScrJP]
		s.stk[nocLaunchStkJP+1] = s.scr[launchScrJP+1]

		// 34   1    Hardware mode
		if s.hwMode, err = rd.ReadByte(); err != nil {
			return err
		}

		if s.addLen == 23 && s.hwMode > 2 {
			s.otek = true // v2 & c>2 then 128k, if v3 then c>3 is 128k
		} else if s.hwMode > 3 {
			s.otek = true
		}

		// 35   1    If in 128 mode, contains last OUT to 0x7ffd
		var lastOut byte
		if lastOut, err = rd.ReadByte(); err != nil {
			return err
		}

		if s.otek {
			s.scr[launchScrOUT] = lastOut
			s.stk[nocLaunchStkOUT] = lastOut
		}

		// 36   1    Contains 0xff if Interface I rom paged [SKIPPED]
		// 37   1    Hardware Modify Byte [SKIPPED]
		if _, err = rd.Discard(2); err != nil {
			return err
		}
		// 38   1    Last OUT to port 0xfffd (soundchip register number)
		if s.bln[mdrBlnFFFD], err = rd.ReadByte(); err != nil {
			return err
		}
		// 39  16    Contents of the sound chip registers
		if _, err := io.ReadFull(rd, s.bln[mdrBlnAY:mdrBlnAY+16]); err != nil {
			return err
		}

		// following is only in v3 snapshots
		// 55   2    Low T state counter [SKIPPED]
		// 57   1    Hi T state counter [SKIPPED]
		// 58   1    Flag byte used by Spectator(QL spec.emulator) [SKIPPED]
		// 59   1    0xff if MGT Rom paged [SKIPPED]
		// 60   1    0xff if Multiface Rom paged.Should always be 0. [SKIPPED]
		// 61   1    0xff if 0 - 8191 is ROM, 0 if RAM [SKIPPED]
		// 62   1    0xff if 8192 - 16383 is ROM, 0 if RAM [SKIPPED]
		// 63  10    5 x keyboard mappings for user defined joystick [SKIPPED]
		// 73  10    5 x ASCII word : keys corresponding to mappings above [SKIPPED]
		// 83   1    MGT type : 0 = Disciple + Epson, 1 = Disciple + HP, 16 = Plus D [SKIPPED]
		// 84   1    Disciple inhibit button status : 0 = out, 0ff = in [SKIPPED]
		// 85   1    Disciple inhibit flag : 0 = rom pageable, 0ff = not [SKIPPED]
		if s.addLen > 23 {
			if _, err = rd.Discard(31); err != nil {
				return err
			}
		}

		// only if version 3 & 55 additional length
		// 86   1    Last OUT to port 0x1ffd, ignored for Microdrive as only
		//           applicable on +3/+2A machines [SKIPPED]
		if s.addLen == 55 {
			if c, err = rd.ReadByte(); err != nil {
				return err
			} else if c&1 == 1 {
				// special page mode so exit as not compatible with
				// earlier 128k machines
				return fmt.Errorf(
					"+3/2A snapshots with special RAM mode enabled not " +
						"supported. Microdrives do not work on +3/+2A hardware.")
			}
		}

		log.WithFields(log.Fields{
			"additional_length": s.addLen,
			"otek":              s.otek,
			"last_out":          lastOut}).Debug("finished extended setup")

	} else {
		log.Debug("finished setup")
	}

	return nil
}

//
func (s *lScreen) setupSNA(rd *bufio.Reader, size int) error {

	if size < 49179 {
		return fmt.Errorf("SNA snapshot issue - too small: %d", size)
	}

	s.otek = size >= 131103 // 128k snapshot

	var c byte
	var err error

	if err = fill(rd, s.bln, []int{
		mdrBlnI,   //		$00  I	Interrupt register
		mdrBlnHLA, //		$01  HL'
		mdrBlnHLA + 1,
		mdrBlnDEA, //		$03  DE'
		mdrBlnDEA + 1,
		mdrBlnBCA, //		$05  BC'
		mdrBlnBCA + 1,
		mdrBlnAFA,     //	$07  F'
		mdrBlnAFA + 1, //	$08  A'
	}); err != nil {
		return err
	}

	if err = fill(rd, s.scr, []int{
		launchScrHL, // 		$09  HL
		launchScrHL + 1,
		launchScrDE, // 		$0B  DE
		launchScrDE + 1,
		launchScrBC, //		$0D  BC
		launchScrBC + 1,
	}); err != nil {
		return err
	}

	s.stk[nocLaunchStkHL] = s.scr[launchScrHL]
	s.stk[nocLaunchStkHL+1] = s.scr[launchScrHL+1]
	s.igp[nocLaunchIgpDE] = s.scr[launchScrDE]
	s.igp[nocLaunchIgpDE+1] = s.scr[launchScrDE+1]
	s.stk[nocLaunchStkBC] = s.scr[launchScrBC]
	s.stk[nocLaunchStkBC+1] = s.scr[launchScrBC+1]

	if err = fill(rd, s.bln, []int{
		mdrBlnIY, //		$0F  IY
		mdrBlnIY + 1,
		mdrBlnIX, //		$11  IX
		mdrBlnIX + 1,
	}); err != nil {
		return err
	}

	// check this is a SNA snapshot
	if (s.bln[mdrBlnI] == 'M' && s.bln[mdrBlnHLA] == 'V' &&
		s.bln[mdrBlnHLA+1] == ' ' && s.bln[mdrBlnDEA] == '-') ||
		(s.bln[mdrBlnI] == 'Z' && s.bln[mdrBlnHLA] == 'X' &&
			s.bln[mdrBlnHLA+1] == '8' && s.bln[mdrBlnDEA] == '2') {
		return fmt.Errorf("not a SNA snapshot")
	}

	//	$13  0 for DI otherwise EI
	if c, err = rd.ReadByte(); err != nil {
		return err
	}

	if c == 0 {
		s.scr[launchScrEI] = 0xf3 // di
		s.stk[nocLaunchStkEI] = 0xf3
	} else {
		s.scr[launchScrEI] = 0xfb // ei
		s.stk[nocLaunchStkEI] = 0xfb
	}

	if err = fill(rd, s.scr, []int{
		launchScrR,      //	$14  R
		launchScrAF,     //	$15  F
		launchScrAF + 1, //	$16  A
		launchScrSP,     //	$17  SP
		launchScrSP + 1,
	}); err != nil {
		return err
	}

	s.stk[nocLaunchStkR] = s.scr[launchScrR]
	s.stk[nocLaunchStkAF] = s.scr[launchScrAF]
	s.stk[nocLaunchStkAF+1] = s.scr[launchScrAF+1]

	s.stkPos = int(s.scr[launchScrSP+1])*256 + int(s.scr[launchScrSP])
	if !s.otek {
		s.stkPos += 2
	}
	if s.stkPos == 0 {
		s.stkPos = 65536
	}
	s.stkPos -= len(nocLaunchStk) // pos of stack code

	p := s.stkPos + nocLaunchStkAF
	s.scr[launchScrSP] = byte(p)
	s.scr[launchScrSP+1] = byte(p >> 8) // start of stack within stack
	s.igp[nocLaunchIgpRD] = byte(p)
	s.igp[nocLaunchIgpRD+1] = byte(p >> 8)

	// $19  Interrupt mode IM(0, 1 or 2)
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	c &= 3

	if c == 0 {
		s.bln[mdrBlnIM] = 0x46 // im 0
	} else if c == 1 {
		s.bln[mdrBlnIM] = 0x56 // im 1
	} else {
		s.bln[mdrBlnIM] = 0x5e // im 2
	}

	//	$1A  Border color
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	c &= 7
	s.bln[mdrBlnBRD] = c + 0x30
	s.bln[mdrBlnPAP] = (c << 3) + c

	return nil
}

//
func (s *lScreen) postSetup(rd *bufio.Reader) (byte, error) {

	if s.otek {

		if err := fill(rd, s.scr, []int{
			launchScrJP, // PC
			launchScrJP + 1,
			launchScrOUT, // last out to 0x7ffd
		}); err != nil {
			return 0, err
		}

		s.stk[nocLaunchStkJP] = s.scr[launchScrJP]
		s.stk[nocLaunchStkJP+1] = s.scr[launchScrJP+1]
		s.stk[nocLaunchStkOUT] = s.scr[launchScrOUT]

		// TD-DOS
		if c, err := rd.ReadByte(); err != nil {
			return 0, err
		} else if c != 0 {
			return 0, fmt.Errorf("SNA snapshot issue")
		}
	}

	return s.scr[launchScrOUT], nil
}

//
func (s *lScreen) adjustStackPos(main []byte, sna bool) (bool, error) {

	stackpos := s.stkPos + len(nocLaunchStk)

	if sna && !s.otek {
		s.scr[launchScrJP] = main[stackpos-16384-2]
		s.scr[launchScrJP+1] = main[stackpos-16384-1]
		s.stk[nocLaunchStkJP] = s.scr[launchScrJP]
		s.stk[nocLaunchStkJP+1] = s.scr[launchScrJP+1]
	}

	if stackpos < 23296 { // stack in screen?

		log.WithField("stack", s.stkPos).Debug("stack in screen")
		i := int(s.scr[launchScrJP+1])*256 + int(s.scr[launchScrJP]) - 16384

		if main[i] == 0x31 { // ld sp,

			// set-up stack
			stackpos = int(main[i+2])*256 + int(main[i+1])
			if stackpos == 0 {
				stackpos = 65536
			}
			s.stkPos = stackpos - len(nocLaunchStk) // pos of stack code
			log.WithField("stack", s.stkPos).Debug("adjusted stack")

			// start of stack within stack
			s.igp[nocLaunchIgpRD] = byte(s.stkPos + nocLaunchStkAF)
			s.igp[nocLaunchIgpRD+1] = byte((s.stkPos + nocLaunchStkAF) >> 8)
			return true, nil
		}

	} else if (s.scr[launchScrOUT]&7) > 0 && stackpos > 49152 && s.otek {
		return false, fmt.Errorf("stack in paged memory won't work")
	}

	return false, nil
}

// does nothing in old launcher
func (s *lScreen) byteSeriesScan(main []byte, delta, dgap int) error {
	return nil
}

//
func (s *lScreen) getAdder(delta, cmSize, maxSize int) (int, error) {

	// sort out adder
	adder := launchScrLen + delta - 3 // add launcher + delta
	maxSize -= delta
	cmSize += adder

	if delta > BGap || cmSize > maxSize {
		return -1, fmt.Errorf("too big to fit in Spectrum memory")
	}

	// BASIC
	launchStart := 16384 // sort out compression start
	s.bln[mdrBlnCPYF] = byte(launchStart)
	s.bln[mdrBlnCPYF+1] = byte(launchStart >> 8)

	s.bln[mdrBlnCPYX] = byte(adder) // how many to copy
	s.bln[mdrBlnCPYX+1] = byte(adder >> 8)

	stack := 16384 + launchScrAF  // change stack
	s.bln[mdrBlnTS] = byte(stack) // stack
	s.bln[mdrBlnTS+1] = byte(stack >> 8)

	start := 16384 + launchScrDELTA
	s.scr[launchScrLCF] = byte(start)
	s.scr[launchScrLCF+1] = byte(start >> 8)
	s.scr[launchScrLCS] = byte(delta) //adjust last copy for delta **fix

	s.bln[mdrBlnJP] = byte(launchStart)
	s.bln[mdrBlnJP+1] = byte(launchStart >> 8)

	// sort out compression start
	start = 65536 - cmSize // compress start
	s.bln[mdrBlnFCPY] = byte(start)
	s.bln[mdrBlnFCPY+1] = byte(start >> 8)

	if !s.isOtek() {
		s.bln[mdrBlnTO] = 0x30 // for i=0 to 0 as only one thing to load
	}

	return adder, nil
}

//
func (s *lScreen) flushRun(r *part) {
	r.data = make([]byte, len(s.bln))
	copy(r.data, s.bln)
	r.length = len(s.bln)
}

//
func (s *lScreen) flushMain(comp, main []byte, adder, delta int) {
	//copy launcher & delta to screen or prtbuff
	copy(comp[8704-adder:], s.scr[:launchScrDELTA])
	copy(comp[8704-adder+launchScrDELTA:], main[49152-delta:49152])
}

// --- 4-Stage launcher -------------------------------------------------------
// this new launcher fixes the problem with screen corruption
//
type lHidden struct {
	lScreen
}

//
func (s *lHidden) byteSeriesScan(main []byte, delta, dgap int) error {

	size := nocLaunchIgpLen + delta - 3
	stack := s.stkPos + len(nocLaunchStk) - 16384 // eq. stackpos - 16384 in C

	log.WithFields(log.Fields{
		"size":  size,
		"stack": s.stkPos,
		"delta": delta,
		"dgap":  dgap}).Debug("byte series scan")

	s.igpPos = 0

	// find maximum gap
	s.vgap = 0x00 // cycle through all bytes
	maxGap := 0
	maxPos := 0
	maxChr := 0

	for {
		j := 0
		for i := 0; i < s.mainSize(); i++ { // also include rest of printer buffer
			ix := i + 6912 + len(nocLaunchPrt)
			if main[ix] == s.vgap {
				j++
				// start of gap > stack, or end of gap < stack - 32, then ok
				if j > maxGap && ((ix-j) > stack || ix < stack-len(nocLaunchStk)) {
					maxGap = j
					maxPos = i + 1
					maxChr = int(s.vgap)
				}
			} else {
				j = 0
			}
		}
		if s.vgap == 0xff {
			break
		}
		s.vgap++
	}

	if maxGap > size {
		s.igpPos = maxPos + 6912 + len(nocLaunchPrt) - maxGap // start of in gap

	} else { // cannot find large enough gap so use screen attr
		s.igpPos = 6912 - size
		vgaps := 0
		vgapb := 0
		for maxChr = 0; maxChr <= 0xff; maxChr++ { //find most common attr
			j := 0
			for i := s.igpPos; i < 6912; i++ {
				if main[i] == byte(maxChr) {
					j++
				}
			}
			if j >= vgapb {
				vgapb = j
				vgaps = maxChr
			}
		}
		maxChr = vgaps
		log.WithFields(log.Fields{
			"vgaps":   vgaps,
			"vgapb":   vgapb,
			"maxchar": maxChr}).Debug("no large enough gap, using screen attr")
	}

	// is pc in the way?
	pc := int(s.stk[nocLaunchStkJP+1])*256 + int(s.stk[nocLaunchStkJP])
	stShift := 0
	if s.stkPos <= pc && s.stkPos+len(nocLaunchStk) > pc {
		stShift = s.stkPos + len(nocLaunchStk) - pc // stack - pc
		if stShift <= 2 {
			return fmt.Errorf("program counter clashes with launcher")
		}
		// shift equivalent of 32 bytes below where is was (4 bytes still
		// remain under the stack)
		stShift = nocLaunchStkAF
	}

	log.WithFields(log.Fields{
		"igppos": s.igpPos, "stshift": stShift}).Debug("byte scan done")

	start := s.igpPos + 16384
	s.prt[nocLaunchPrtJP] = byte(start)
	s.prt[nocLaunchPrtJP+1] = byte(start >> 8) // jump into gap

	start = s.igpPos + nocLaunchIgpBEGIN + 16384
	s.igp[nocLaunchIgpBDATA] = byte(start)
	s.igp[nocLaunchIgpBDATA+1] = byte(start >> 8) // bdata start
	s.igp[nocLaunchIgpLCS] = byte(delta)

	if size == 256 {
		s.igp[nocLaunchIgpCLR] = 0
	} else {
		s.igp[nocLaunchIgpCLR] = byte(size) // size of ingap clear
	}

	s.igp[nocLaunchIgpCHR] = byte(maxChr) // set the erase char in stack code
	start = s.stkPos - stShift
	s.igp[nocLaunchIgpJP] = byte(start)
	s.igp[nocLaunchIgpJP+1] = byte(start >> 8) // jump to stack code - adjust - shift

	// copy stack routine under stack, split version if shift
	if stShift > 0 {
		copy(main[s.stkPos-16384-stShift:], s.stk[:len(nocLaunchStk)-2])
		// final 2 bytes just below new code
		for i := 0; i < 2; i++ {
			main[s.stkPos+len(nocLaunchStk)-16384+i-2] = s.stk[len(nocLaunchStk)-2+i]
		}
	} else {
		if s.stkPos < 16384 {
			return fmt.Errorf(
				"corrupted snapshot data - stack too low: %d", s.stkPos)
		}
		copy(main[s.stkPos-16384:], s.stk[:len(nocLaunchStk)]) // standard copy
	}

	// if ingap not in screen attr, this is done after so as to not effect
	// the screen compression
	if s.igpPos >= 6912 {
		// copy prtbuf to code
		copy(main[s.igpPos+nocLaunchIgpBEGIN+delta:],
			main[6912:6912+len(nocLaunchPrt)])
		// copy delta to code
		copy(main[s.igpPos+nocLaunchIgpBEGIN:], main[49152-delta:49152])
		// copy in compression routine into main
		copy(main[s.igpPos:], s.igp[:nocLaunchIgpBEGIN])
	}

	return nil
}

//
func (s *lHidden) getAdder(delta, cmSize, maxSize int) (int, error) {

	// sort out adder
	adder := len(nocLaunchPrt) // just add prtbuf launcher
	if s.igpPos < 6912 {       // if ingap in screen
		adder += len(nocLaunchPrt) + delta + nocLaunchIgpBEGIN
	}

	maxSize -= delta
	cmSize += adder

	if delta > BGap || cmSize > maxSize {
		return -1, fmt.Errorf("too big to fit in Spectrum memory")
	}

	// BASIC
	launchStart := 23296 + 2
	if s.igpPos < 6912 { // sort out compression start
		start := 23296 + len(nocLaunchPrt) - adder // compress start
		s.bln[mdrBlnCPYF] = byte(start)
		s.bln[mdrBlnCPYF+1] = byte(start >> 8)
		s.bln[mdrBlnCPYX] = byte(adder)
		s.bln[mdrBlnCPYX+1] = byte(adder >> 8)
	}

	s.bln[mdrBlnJP] = byte(launchStart)
	s.bln[mdrBlnJP+1] = byte(launchStart >> 8)

	// sort out compression start
	start := 65536 - cmSize // compress start
	s.bln[mdrBlnFCPY] = byte(start)
	s.bln[mdrBlnFCPY+1] = byte(start >> 8)

	if !s.isOtek() {
		s.bln[mdrBlnTO] = 0x30 // for i=0 to 0 as only one thing to load
	}

	return adder, nil
}

//
func (s *lHidden) flushMain(comp, main []byte, adder, delta int) {

	if s.igpPos < 6912 {
		// copy prtbuf to ingap code
		copy(comp[8704-adder:], s.igp[:nocLaunchIgpBEGIN])
		copy(comp[8704-adder+nocLaunchIgpBEGIN:], main[49152-delta:49152])
		copy(comp[8704-adder+nocLaunchIgpBEGIN+delta:],
			main[6912:6912+len(nocLaunchPrt)])
	}

	copy(comp[8704-len(s.prt):], s.prt)
}

//
func (s *lHidden) startPos() int {
	return 6966 // 0x5b36 onward so have to save at least 1562 bytes
}

//
func (s *lHidden) mainSize() int {
	return 42186
}

//
func validateHardwareMode(mode byte, version int) (string, error) {

	hw := ""
	supported := true

	switch mode {

	case 0:
		hw = "48k"
	case 1:
		hw = "48k + If.1"
	case 2:
		hw = "SamRam"
		supported = false
	case 3:
		if version == 2 {
			hw = "128k"
		} else {
			hw = "48k + M.G.T."
		}
	case 4:
		if version == 2 {
			hw = "128k + If.1"
		} else {
			hw = "128k"
		}
	case 5:
		if version == 3 {
			hw = "128k + If.1"
		}
	case 6:
		if version == 3 {
			hw = "128k + M.G.T."
		}
	case 7:
		hw = "Spectrum +3"
	case 8:
		hw = "Spectrum +3 (incorrect)"
	case 9:
		hw = "Pentagon (128K)"
	case 10:
		hw = "Scorpion (256K)"
		supported = false
	case 11:
		hw = "Didaktik-Kompakt"
		supported = false
	case 12:
		hw = "Spectrum +2"
	case 13:
		hw = "Spectrum +2A"
	case 14:
		hw = "TC2048"
		supported = false
	case 15:
		hw = "TC2068"
		supported = false
	case 128:
		hw = "TS2068"
		supported = false
	}

	if hw == "" {
		return "", fmt.Errorf("invalid h/w mode: %d", mode)
	}

	if !supported {
		return "", fmt.Errorf("unsupported h/w mode: %s (%d)", hw, mode)
	}

	return hw, nil
}
