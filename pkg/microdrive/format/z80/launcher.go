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

	log.Debugf("using launcher type '%s'", typ)
	return l, nil
}

//
type launcher interface {
	setup(rd *bufio.Reader, sna bool) error
	byteSeriesScan(main []byte, delta, dgap int) error
	flush(main []byte, launch *part, delta int)
	get(ix int) byte
	set(ix int, b byte)
	getData() []byte
	isCompressed() bool
	hardwareMode() byte
	isOtek() bool
	borderColor() byte
	addLength() int
	randomize() int
	startPos() int
	// stackPos is the absolute stack pointer int the 64k address space;
	// it corresponds to `noc_launchstk_pos` in the C version, and is
	// `len(nocLaunchStk)` lower than `stackpos` from the C version
	stackPos() int
	adjustStackPos(main []byte, sna bool) bool
	mainSize() int
}

// --- screen based launcher --------------------------------------------------
// usually exhibits some amount of screen corruption at top of screen
//
type lScreen struct {
	data       []byte
	compressed bool
	hiBitR     bool
	otek       bool
	border     byte
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
func (s *lScreen) borderColor() byte {
	return s.border
}

//
func (s *lScreen) addLength() int {
	return s.addLen
}

//
func (s *lScreen) randomize() int {
	return 16384
}

//
func (s *lScreen) startPos() int {
	return 6912
}

//
func (s *lScreen) stackPos() int {
	return s.stkPos
}

//
func (s *lScreen) mainSize() int {
	return 42240
}

//
func (s *lScreen) getData() []byte {
	return s.data
}

//
func (s *lScreen) get(ix int) byte {
	return s.data[ix]
}

//
func (s *lScreen) set(ix int, b byte) {
	s.data[ix] = b
}

//
func (s *lScreen) setup(rd *bufio.Reader, sna bool) error {

	s.data = make([]byte, len(launchMDRFull))
	copy(s.data, launchMDRFull)

	if sna {
		return s.setupSNA(rd)
	} else {
		return s.setupZ80(rd)
	}
}

//
func (s *lScreen) setupZ80(rd *bufio.Reader) error {

	var c byte
	var err error

	if err = fill(rd, s.data, []int{ // read in z80 starting with header
		ixA,  //      0   1    A register
		ixIF, //      1   1    F register
		ixBC, //      2   2    BC register pair(LSB, i.e.C, first)
		ixBC + 1,
		ixHL, //      4   2    HL register pair
		ixHL + 1,
		ixJP, //      6   2    Program counter (if zero then version 2 or 3 snapshot)
		ixJP + 1,
		ixSP, //      8   2    Stack pointer
		ixSP + 1,
		ixIF + 1, // 10   1    Interrupt register
		ixR,      // 11   1    Refresh register (Bit 7 is not significant!)
	}); err != nil {
		return err
	}

	// pos of stack code
	if s.stkPos = int(s.data[ixSP+1])*256 + int(s.data[ixSP]); s.stkPos == 0 {
		s.stkPos = 65536
	}
	s.stkPos -= len(nocLaunchStk)

	// r, reduce by 6 so correct on launch
	if err = adjust(s.data, ixR, -6); err != nil {
		return err
	}

	//  12   1    Bit 0: Bit 7 of r register; Bit 1-3: Border colour;
	//            Bit 4=1: SamROM; Bit 5=1:v1 Compressed; Bit 6-7: N/A
	if c, err = rd.ReadByte(); err != nil {
		return err
	}

	s.compressed = (c&32)>>5 == 1 // 1 compressed, 0 not
	s.hiBitR = c&1 == 1 || c > 127

	if s.hiBitR {
		s.data[ixR] = s.data[ixR] | 128 // r high bit set
	} else {
		s.data[ixR] = s.data[ixR] & 127 // r high bit reset
	}

	s.border = ((c & 14) >> 1) + 0x30 //border/paper col

	if err = fill(rd, s.data, []int{
		ixDE, //      13   2    DE register pair
		ixDE + 1,
		ixBCA, //     15   2    BC' register pair
		ixBCA + 1,
		ixDEA, //     17   2    DE' register pair
		ixDEA + 1,
		ixHLA, //     19   2    HL' register pair
		ixHLA + 1,
		ixAFA + 1, // 21   1    A' register
		ixAFA,     // 22   1    F' register
		ixIY,      // 23   2    IY register (Again LSB first)
		ixIY + 1,
		ixIX, //      25   2    IX register
		ixIX + 1,
	}); err != nil {
		return err
	}

	// 27   1    Interrupt flip flop, 0 = DI, otherwise EI
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	if c == 0 {
		s.data[ixEI] = 0xf3 // di
	} else {
		s.data[ixEI] = 0xfb // ei
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
		s.data[ixIM] = 0x46 // im 0
	} else if c == 1 {
		s.data[ixIM] = 0x56 // im 1
	} else {
		s.data[ixIM] = 0x5e // im 2
	}

	// version 2 & 3 only
	s.addLen = 0 // 0 indicates v1, 23 for v2 otherwise v3
	s.otek = false

	if s.data[ixJP] == 0 && s.data[ixJP+1] == 0 {

		// 30   2    Length of additional header block
		if s.addLen, err = readUInt16(rd); err != nil {
			return err
		}
		log.Debugf("additional length: %d", s.addLen)

		// 32   2    Program counter
		if err = fill(rd, s.data, []int{ixJP, ixJP + 1}); err != nil {
			return err
		}

		// 34   1    Hardware mode
		if s.hwMode, err = rd.ReadByte(); err != nil {
			return err
		}

		if s.addLen == 23 && s.hwMode > 2 {
			s.otek = true // v2 & c>2 then 128k, if v3 then c>3 is 128k
		} else if s.hwMode > 3 {
			s.otek = true
		}
		log.Debugf("otek: %v", s.otek)

		// 35   1    If in 128 mode, contains last OUT to 0x7ffd
		if c, err = rd.ReadByte(); err != nil {
			return err
		}
		log.Debugf("last out: %d", c)

		if s.otek {
			s.data[ixOUT] = c
		}

		// 36   1    Contains 0xff if Interface I rom paged [SKIPPED]
		// 37   1    Hardware Modify Byte [SKIPPED]
		// 38   1    Last OUT to port 0xfffd (soundchip register number) [SKIPPED]
		// 39  16    Contents of the sound chip registers [SKIPPED] *ideally for
		//           128k setting ay registers make sense, however in practise
		//           never found it is needed
		if _, err = rd.Discard(19); err != nil {
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
	}

	return nil
}

//
func (s *lScreen) setupSNA(rd *bufio.Reader) error {

	var c byte
	var err error

	if err = fill(rd, s.data, []int{
		ixIF + 1, //	$00  I	Interrupt register
		ixHLA,    //	$01  HL'
		ixHLA + 1,
		ixDEA, //		$03  DE'
		ixDEA + 1,
		ixBCA, //		$05  BC'
		ixBCA + 1,
		ixAFA,     //	$07  F'
		ixAFA + 1, //	$08  A'
		ixHL,      //	$09  HL
		ixHL + 1,
		ixDE, //		$0B  DE
		ixDE + 1,
		ixBC, //		$0D  BC
		ixBC + 1,
		ixIY, //		$0F  IY
		ixIY + 1,
		ixIX, //		$11  IX
		ixIX + 1,
	}); err != nil {
		return err
	}

	//	$13  0 for DI otherwise EI
	if c, err = rd.ReadByte(); err != nil {
		return err
	}

	if c == 0 {
		s.data[ixEI] = 0xf3 //di
	} else {
		s.data[ixEI] = 0xfb //ei
	}

	if err = fill(rd, s.data, []int{
		ixR,  //	$14  R
		ixIF, //	$15  F
		ixA,  //	$16  A
		ixSP, //	$17  SP
		ixSP + 1,
	}); err != nil {
		return err
	}

	// pos of stack code
	if s.stkPos = int(s.data[ixSP+1])*256 + int(s.data[ixSP]) + 2; s.stkPos == 0 {
		s.stkPos = 65536
	}
	s.stkPos -= len(nocLaunchStk)

	// $19  Interrupt mode IM(0, 1 or 2)
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	c &= 3

	if c == 0 {
		s.data[ixIM] = 0x46 //im 0
	} else if c == 1 {
		s.data[ixIM] = 0x56 //im 1
	} else {
		s.data[ixIM] = 0x5e //im 2
	}

	//	$1A  Border color
	if c, err = rd.ReadByte(); err != nil {
		return err
	}
	s.border = (c & 7) + 0x30

	return nil
}

//
func (s *lScreen) adjustStackPos(main []byte, sna bool) bool {

	stackpos := s.stkPos + len(nocLaunchStk)

	if sna {
		s.data[ixJP] = main[stackpos-16384-2]
		s.data[ixJP+1] = main[stackpos-16384-1]
	}

	if stackpos < 23296 { // stack in screen?
		log.WithField("stack", s.stkPos).Debug("stack in screen")
		i := int(s.get(ixJP+1))*256 + int(s.get(ixJP)) - 16384
		if main[i] == 0x31 { // ld sp,
			// set-up stack
			s.stkPos = int(main[i+2])*256 + int(main[i+1])
			if s.stkPos == 0 {
				s.stkPos = 65536
			}
			s.stkPos -= len(nocLaunchStk) // pos of stack code
			log.WithField("stack", s.stkPos).Debug("adjusted stack")
			return true
		}
	}

	return false
}

// does nothing in old launcher
func (s *lScreen) byteSeriesScan(main []byte, delta, dgap int) error {
	return nil
}

//
func (s *lScreen) flush(main []byte, launch *part, delta int) {

	s.data[ixLCS] = byte(delta) //adjust last copy for delta

	//copy end delta*bytes to launcher
	copy(s.data[launchMDRFullLen:], main[49152-delta:])

	launch.length = launchMDRFullLen + delta
	launch.start = 16384
	launch.data = s.data
}

// --- 3-Stage launcher -------------------------------------------------------
// this new launcher fixes the problem with screen corruption
//
type lHidden struct {
	lScreen
	//
	prt []byte
	igp []byte
	stk []byte
	//
	igpPos    int
	vgap      byte
	pageShift int
}

//
func (s *lHidden) setup(rd *bufio.Reader, sna bool) error {

	if err := s.lScreen.setup(rd, sna); err != nil {
		return err
	}

	s.prt = make([]byte, len(nocLaunchPrt))
	copy(s.prt, nocLaunchPrt)
	s.igp = make([]byte, len(nocLaunchIgp))
	copy(s.igp, nocLaunchIgp)
	s.stk = make([]byte, len(nocLaunchStk))
	copy(s.stk, nocLaunchStk)

	if sna {
		return s.setupSNA()
	} else {
		return s.setupZ80()
	}
}

//
func (s *lHidden) setupZ80() error {

	s.stk[nocLaunchStkA] = s.data[ixA]
	s.stk[nocLaunchStkIF] = s.data[ixIF]
	s.stk[nocLaunchStkBC] = s.data[ixBC]
	s.stk[nocLaunchStkBC+1] = s.data[ixBC+1]
	s.stk[nocLaunchStkHL] = s.data[ixHL]
	s.stk[nocLaunchStkHL+1] = s.data[ixHL+1]
	s.stk[nocLaunchStkJP] = s.data[ixJP]
	s.stk[nocLaunchStkJP+1] = s.data[ixJP+1]

	s.igp[nocLaunchIgpRD] = byte(s.stkPos + nocLaunchStkAFA)
	s.igp[nocLaunchIgpRD+1] = byte((s.stkPos + nocLaunchStkAFA) >> 8) // start of stack within stack

	s.stk[nocLaunchStkIF+1] = s.data[ixIF+1]
	s.stk[nocLaunchStkR] = s.data[ixR] + 1 // 5 for 3 stage launcher

	if s.hiBitR {
		s.stk[nocLaunchStkR] |= 128 // r high bit set
	} else {
		s.stk[nocLaunchStkR] &= 127 //r high bit reset
	}

	s.igp[nocLaunchIgpDE] = s.data[ixDE]
	s.igp[nocLaunchIgpDE+1] = s.data[ixDE+1]
	s.igp[nocLaunchIgpBCA] = s.data[ixBCA]
	s.igp[nocLaunchIgpBCA+1] = s.data[ixBCA+1]
	s.igp[nocLaunchIgpDEA] = s.data[ixDEA]
	s.igp[nocLaunchIgpDEA+1] = s.data[ixDEA+1]
	s.igp[nocLaunchIgpHLA] = s.data[ixHLA]
	s.igp[nocLaunchIgpHLA+1] = s.data[ixHLA+1]
	s.stk[nocLaunchStkAFA+1] = s.data[ixAFA+1]
	s.stk[nocLaunchStkAFA] = s.data[ixAFA]
	s.igp[nocLaunchIgpIY] = s.data[ixIY]
	s.igp[nocLaunchIgpIY+1] = s.data[ixIY+1]
	s.igp[nocLaunchIgpIX] = s.data[ixIX]
	s.igp[nocLaunchIgpIX+1] = s.data[ixIX+1]
	s.stk[nocLaunchStkEI] = s.data[ixEI]
	s.stk[nocLaunchStkIM] = s.data[ixIM]

	if s.otek {
		s.igp[nocLaunchIgpOUT] = s.data[ixOUT]
	}

	return nil
}

//
func (s *lHidden) setupSNA() error {

	s.stk[nocLaunchStkIF+1] = s.data[ixIF+1]

	s.igp[nocLaunchIgpHLA] = s.data[ixHLA]
	s.igp[nocLaunchIgpHLA+1] = s.data[ixHLA+1]
	s.igp[nocLaunchIgpDEA] = s.data[ixDEA]
	s.igp[nocLaunchIgpDEA+1] = s.data[ixDEA+1]
	s.igp[nocLaunchIgpBCA] = s.data[ixBCA]
	s.igp[nocLaunchIgpBCA+1] = s.data[ixBCA+1]

	s.stk[nocLaunchStkAFA] = s.data[ixAFA]
	s.stk[nocLaunchStkAFA+1] = s.data[ixAFA+1]
	s.stk[nocLaunchStkHL] = s.data[ixHL]
	s.stk[nocLaunchStkHL+1] = s.data[ixHL+1]

	s.igp[nocLaunchIgpDE] = s.data[ixDE]
	s.igp[nocLaunchIgpDE+1] = s.data[ixDE+1]

	s.stk[nocLaunchStkBC] = s.data[ixBC]
	s.stk[nocLaunchStkBC+1] = s.data[ixBC+1]

	s.igp[nocLaunchIgpIY] = s.data[ixIY]
	s.igp[nocLaunchIgpIY+1] = s.data[ixIY+1]
	s.igp[nocLaunchIgpIX] = s.data[ixIX]
	s.igp[nocLaunchIgpIX+1] = s.data[ixIX+1]

	s.stk[nocLaunchStkEI] = s.data[ixEI]
	s.stk[nocLaunchStkIF] = s.data[ixIF]
	s.stk[nocLaunchStkA] = s.data[ixA]
	s.stk[nocLaunchStkIM] = s.data[ixIM]

	return nil
}

//
func (s *lHidden) adjustStackPos(main []byte, sna bool) bool {

	adjusted := s.lScreen.adjustStackPos(main, sna)

	if sna {
		s.stk[nocLaunchStkJP] = s.data[ixJP]
		s.stk[nocLaunchStkJP+1] = s.data[ixJP+1]
	}

	if sna || adjusted {
		// start of stack within stack
		s.igp[nocLaunchIgpRD] = byte(s.stkPos + nocLaunchStkAFA)
		s.igp[nocLaunchIgpRD+1] = byte((s.stkPos + nocLaunchStkAFA) >> 8)
	}

	return adjusted
}

//
func (s *lHidden) byteSeriesScan(main []byte, delta, dgap int) error {

	if s.pageShift == 0 { // only set up once
		s.pageShift = 42173 // 49152-6912-67
		// bits 0-2: RAM page (0-7) to map into memory at 0Xc000
		if s.igp[nocLaunchIgpOUT]&7 > 0 {
			s.pageShift -= 16384
		}
	}

	size := nocLaunchIgpLen + delta - 3
	stack := s.stkPos + len(nocLaunchStk) - 16384

	log.WithFields(log.Fields{
		"size":      size,
		"stack":     s.stkPos,
		"delta":     delta,
		"dgap":      dgap,
		"pageshift": s.pageShift}).Debug("byte series scan")

	s.igpPos = 0

	// find gap
	s.vgap = 0x00 // cycle through all bytes
	maxGap := 0
	maxPos := 0
	maxChr := 0

	for {
		j := 0
		for i := 0; i < s.pageShift; i++ { // also include rest of printer buffer
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

	adjust := 0 // start with no adjustments between ingap and stack
	if maxGap > size {
		s.igpPos = maxPos + 6912 + len(nocLaunchPrt) - maxGap // start of in gap

	} else { // cannot find large enough gap so can we adjust the launcher?
		for _, a := range adjGap {
			if maxGap > size-int(a) {
				adjust = int(a)
			}
			if adjust != 0 {
				break
			}
		}

		if adjust == 0 { // if cannot adjust and not gap big enough then use screen attr
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
			// FIXME
			log.Debugf("vgaps: %d, vgapb: %d, maxchar: %d", vgaps, vgapb, maxChr)
		} else { // can adjust to get gap to fit pos
			s.igpPos = maxPos + 6912 + len(nocLaunchPrt) - maxGap // adjust
		}
	}

	// is pc in the way?
	pc := int(s.stk[nocLaunchStkJP+1])*256 + int(s.stk[nocLaunchStkJP])
	stShift := 0
	if s.stkPos-adjust <= pc && s.stkPos+len(nocLaunchStk) > pc {
		stShift = s.stkPos + len(nocLaunchStk) - pc // stack - pc
		if stShift <= 4 {
			return fmt.Errorf("program counter clashes with launcher")
		}
		// shift equivalent of 32 bytes below where is was (4 bytes still
		// remain under the stack)
		stShift = 28
	}

	log.WithFields(log.Fields{
		"adjust":  adjust,
		"igppos":  s.igpPos,
		"stshift": stShift}).Debug("byte scan done")

	start := s.igpPos + 16384
	s.prt[nocLaunchPrtJP] = byte(start)
	s.prt[nocLaunchPrtJP+1] = byte(start >> 8) // jump into gap

	start = s.igpPos + nocLaunchIgpBEGIN + 16384 - adjust
	s.igp[nocLaunchIgpBDATA] = byte(start)
	s.igp[nocLaunchIgpBDATA+1] = byte(start >> 8) // bdata start
	s.igp[nocLaunchIgpLCS] = byte(delta)

	s.igp[nocLaunchIgpCLR] = byte(size - adjust) // size of ingap clear
	s.igp[nocLaunchIgpCHR] = byte(maxChr)        // set the erase char in stack code
	start = s.stkPos - adjust - stShift
	s.igp[nocLaunchIgpJP] = byte(start)
	s.igp[nocLaunchIgpJP+1] = byte(start >> 8) // jump to stack code - adjust - shift

	// copy stack routine under stack, split version if shift
	if stShift > 0 {
		copy(main[s.stkPos-16384-stShift:], s.stk[:len(nocLaunchStk)-4])
		// final 4 bytes just below new code
		for i := 0; i < 4; i++ {
			main[s.stkPos+len(nocLaunchStk)-16384+i-4] = s.stk[len(nocLaunchStk)-4+i]
		}
	} else {
		if s.stkPos < 16384 {
			return fmt.Errorf(
				"corrupted snapshot data - stack too low: %d", s.stkPos)
		}
		copy(main[s.stkPos-16384:], s.stk[:len(nocLaunchStk)]) // standard copy
	}

	//reduce ingap code and add to stack routine
	if adjust > 0 {
		for i := 0; i < adjust; i++ {
			main[s.stkPos-16384+i-adjust-stShift] = s.igp[nocLaunchIgpBEGIN-adjust-3+i]
		}
	}
	// if ingap not in screen attr, this is done after so as to not effect
	// the screen compression

	if s.igpPos > 6912 {
		// copy prtbuf to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN+delta-adjust:],
			main[6912:6912+len(nocLaunchPrt)])
		// copy delta to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN-adjust:], main[49152-delta:49152])
		// copy in compression routine into screen
		copy(main[s.igpPos:], s.igp[:nocLaunchIgpBEGIN-adjust-3])
		// last jp
		for i := 0; i < 3; i++ {
			main[s.igpPos+nocLaunchIgpBEGIN-adjust-3+i] = s.igp[nocLaunchIgpBEGIN-3+i]
		}
	}

	return nil
}

//
func (s *lHidden) flush(main []byte, launch *part, delta int) {

	s.prt[nocLaunchPrtCP] = s.data[ixCP]
	s.prt[nocLaunchPrtCP+1] = s.data[ixCP+1]

	launch.start = 23296
	launch.length = 256
	startpos := 6912

	if s.igpPos < 6912 {
		// copy prtbuf to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN+delta:],
			main[6912:6912+len(nocLaunchPrt)])
		// copy delta to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN:], main[49152-delta:49152])
		// copy in compression routine into screen
		copy(main[s.igpPos:], s.igp[:nocLaunchIgpBEGIN])

		launch.start = 23296 - (nocLaunchIgpLen + delta - 3)
		launch.length = 256 + (nocLaunchIgpLen + delta - 3)
		startpos -= (nocLaunchIgpLen + delta - 3)
	}

	// copy experimental loader into prtbuf
	copy(main[6912:], s.prt[:len(nocLaunchPrt)])

	launch.data = main[startpos:]
}

//
func (s *lHidden) randomize() int {
	return s.lScreen.randomize() + 6912
}

//
func (s *lHidden) startPos() int {
	return s.lScreen.startPos() + 256
}

//
func (s *lHidden) mainSize() int {
	return s.lScreen.mainSize() - 256
}

//
func validateHardwareMode(mode byte, version int) error {

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
		return fmt.Errorf("invalid h/w mode: %d", mode)
	}

	if !supported {
		return fmt.Errorf("unsupported h/w mode: %s (%d)", hw, mode)
	}

	log.Debugf("h/w mode: %s (%d)", hw, mode)
	return nil
}
