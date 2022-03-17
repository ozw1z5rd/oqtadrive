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

	log.Debugf("using Z80 launcher type '%s'", typ)
	return l, nil
}

//
type launcher interface {
	setup(rd *bufio.Reader) error
	byteSeriesScan(main []byte, delta, dgap int)
	flush(main []byte, launch *part, delta int)
	set(ix int, b byte)
	getData() []byte
	isCompressed() bool
	isOtek() bool
	borderColor() byte
	addLength() int
	randomize() int
	startPos() int
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
}

//
func (s *lScreen) isCompressed() bool {
	return s.compressed
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
func (s *lScreen) mainSize() int {
	return 42240
}

//
func (s *lScreen) getData() []byte {
	return s.data
}

//
func (s *lScreen) set(ix int, b byte) {
	s.data[ix] = b
}

//
func (s *lScreen) setup(rd *bufio.Reader) error {

	var c byte
	var err error

	s.data = make([]byte, len(launchMDRFull))
	copy(s.data, launchMDRFull)

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
		if c, err = rd.ReadByte(); err != nil {
			return err
		}
		log.Debugf("h/w mode: %d", c)

		if c == 2 {
			return fmt.Errorf("SamRAM Z80 snapshots not supported")
		}
		if s.addLen == 23 && c > 2 {
			s.otek = true // v2 & c>2 then 128k, if v3 then c>3 is 128k
		} else if c > 3 {
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

// does nothing in old launcher
func (s *lScreen) byteSeriesScan(main []byte, delta, dgap int) {}

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
	stkPos int
	igpPos int
	vgap   byte
}

//
func (s *lHidden) setup(rd *bufio.Reader) error {

	if err := s.lScreen.setup(rd); err != nil {
		return err
	}

	s.prt = make([]byte, len(nocLaunchPrt))
	copy(s.prt, nocLaunchPrt)
	s.igp = make([]byte, len(nocLaunchIgp))
	copy(s.igp, nocLaunchIgp)
	s.stk = make([]byte, len(nocLaunchStk))
	copy(s.stk, nocLaunchStk)

	s.stk[nocLaunchStkA] = s.data[ixA]
	s.stk[nocLaunchStkIF] = s.data[ixIF]
	s.stk[nocLaunchStkBC] = s.data[ixBC]
	s.stk[nocLaunchStkBC+1] = s.data[ixBC+1]
	s.stk[nocLaunchStkHL] = s.data[ixHL]
	s.stk[nocLaunchStkHL+1] = s.data[ixHL+1]
	s.stk[nocLaunchStkJP] = s.data[ixJP]
	s.stk[nocLaunchStkJP+1] = s.data[ixJP+1]

	// pos of stack code
	s.stkPos = int(s.data[ixSP+1])*256 + int(s.data[ixSP]) - len(nocLaunchStk)

	s.igp[nocLaunchIgpJP] = byte(s.stkPos)
	s.igp[nocLaunchIgpJP+1] = byte(s.stkPos >> 8) // jump to stack code
	s.stk[nocLaunchStkRD] = byte(s.stkPos + 47)
	s.stk[nocLaunchStkRD+1] = byte((s.stkPos + 47) >> 8) // start of stack within stack

	s.stk[nocLaunchStkIF+1] = s.data[ixIF+1]
	s.stk[nocLaunchStkR] = s.data[ixR] - 1 // 5 for 3 stage launcher

	if s.hiBitR {
		s.stk[nocLaunchStkR] |= 128 // r high bit set
	} else {
		s.stk[nocLaunchStkR] &= 127 //r high bit reset
	}

	s.stk[nocLaunchStkDE] = s.data[ixDE]
	s.stk[nocLaunchStkDE+1] = s.data[ixDE+1]
	s.stk[nocLaunchStkBCA] = s.data[ixBCA]
	s.stk[nocLaunchStkBCA+1] = s.data[ixBCA+1]
	s.stk[nocLaunchStkDEA] = s.data[ixDEA]
	s.stk[nocLaunchStkDEA+1] = s.data[ixDEA+1]
	s.stk[nocLaunchStkHLA] = s.data[ixHLA]
	s.stk[nocLaunchStkHLA+1] = s.data[ixHLA+1]
	s.stk[nocLaunchStkAFA+1] = s.data[ixAFA+1]
	s.stk[nocLaunchStkAFA] = s.data[ixAFA]
	s.stk[nocLaunchStkIY] = s.data[ixIY]
	s.stk[nocLaunchStkIY+1] = s.data[ixIY+1]
	s.stk[nocLaunchStkIX] = s.data[ixIX]
	s.stk[nocLaunchStkIX+1] = s.data[ixIX+1]
	s.stk[nocLaunchStkEI] = s.data[ixEI]
	s.stk[nocLaunchStkIM] = s.data[ixIM]

	if s.otek {
		s.stk[nocLaunchStkOUT] = s.data[ixOUT]
	}

	return nil
}

//
func (s *lHidden) byteSeriesScan(main []byte, delta, dgap int) {

	size := nocLaunchIgpLen + delta - 3
	stack := s.stkPos - 16384

	log.Debugf("byte series scan: size %d, stack: %d, delta: %d, dgap: %d",
		size, s.stkPos, delta, dgap)

	if s.igpPos > 0 { // if delta+1 loop then clear area first
		log.Debugf("clearing area with vgap %d", s.vgap)
		// -1 as delta just increased by dgap
		for i := 0; i < size-dgap; i++ {
			main[s.igpPos+i] = s.vgap
		}
	}
	s.igpPos = 0

	// find gap
	s.vgap = 0x00 // cycle through all bytes
	for {
		j := 0
		for i := 0; i < 41984; i++ {
			ix := i + 6912 + 256
			if main[ix] == s.vgap {
				j++
			} else {
				j = 0
			}
			// start of gap > stack then ok, or end of gap < stack - 67 then ok
			if j >= size && (ix-j > stack || ix < stack-67) {
				s.igpPos = ix - j // start of storage
				break
			}
		}
		if s.igpPos > 0 {
			break
		}
		if s.vgap == 0xff {
			break
		}
		s.vgap++
	}

	if s.igpPos > 0 {
		log.Debugf("found gap at %d", s.igpPos)
	} else {
		log.Debugf("no gap found")
		// no space so use attr space instead
		s.igpPos = 6912 - size
		vgaps := 0 //find best vgap
		vgapb := 0
		s.vgap = 0x00
		for {
			j := 0
			for i := s.igpPos; i < 6912; i++ {
				if main[i] == s.vgap {
					j++
				}
			}
			if j >= vgapb {
				vgapb = j
				vgaps = int(s.vgap)
			}
			if s.vgap == 0xff {
				break
			}
			s.vgap++
		}
		s.vgap = byte(vgaps)
		log.Debugf("using attr space with igpPos at %d", s.igpPos)
	}

	log.Debugf("byte scan done")

	start := s.igpPos + 16384
	s.prt[nocLaunchPrtJP] = byte(start)
	s.prt[nocLaunchPrtJP+1] = byte(start >> 8) // jump into gap

	start = s.igpPos + nocLaunchIgpBEGIN + 16384
	s.igp[nocLaunchIgpBDATA] = byte(start)
	s.igp[nocLaunchIgpBDATA+1] = byte(start >> 8) // bdata start
	s.igp[nocLaunchIgpLCS] = byte(delta)

	s.stk[nocLaunchStkCLR] = byte(size)
	s.stk[nocLaunchStkCHR] = s.vgap // set the erase char in stack code

	// copy stack routine under stack
	copy(main[s.stkPos-16384:], s.stk[:len(nocLaunchStk)])

	if s.igpPos > 6912 {
		// copy prtbuf to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN+delta:],
			main[6912:6912+len(nocLaunchPrt)])
		// copy delta to screen
		copy(main[s.igpPos+nocLaunchIgpBEGIN:], main[49152-delta:49152])
		// copy in compression routine into screen
		copy(main[s.igpPos:], s.igp[:nocLaunchIgpBEGIN])
	}
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
