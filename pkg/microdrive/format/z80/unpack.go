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
	"bufio"
	"bytes"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

//
func (s *snapshot) unpack(in io.Reader) error {

	// to determine size, we need to read in all and repack as buffer reader
	buf, err := io.ReadAll(io.LimitReader(in, 256000))
	if err != nil {
		return err
	}

	rd := bufio.NewReader(bytes.NewReader(buf))

	if err := s.launcher.setup(rd, s.sna, len(buf)); err != nil {
		return err
	}

	// space for decompression of z80
	// 8 * 16384 = 131072 bytes
	// 0 - 49152 - Pages 5,2 & 0 (main memory)
	// *128k only - 49152 -  65536: Page 1
	//              65536 -  81920: Page 3
	//              81920 -  98304: Page 4
	//              98304 - 114688: Page 6
	//             114688 - 131072: Page 7
	//
	fullSize := 49152
	if s.launcher.isOtek() {
		fullSize = 131072
	}
	s.main = make([]byte, fullSize)

	// which version of z80?
	length := 0
	s.bank = make([]int, 11)

	for i := range s.bank {
		s.bank[i] = 99 // default
	}

	// set-up bank locations
	var bankEnd int
	if s.launcher.isOtek() {
		s.bank[3] = 32768   //page 0
		s.bank[4] = 49152   //page 1
		s.bank[5] = 16384   //page 2
		s.bank[6] = 65536   //page 3
		s.bank[7] = 81920   //page 4
		s.bank[8] = 0       //page 5
		s.bank[9] = 98304   //page 6
		s.bank[10] = 114688 //page 7
		bankEnd = 8
	} else {
		s.bank[4] = 16384 //page 2
		s.bank[5] = 32768 //page 0
		s.bank[8] = 0     //page 5
		bankEnd = 3
	}

	if s.launcher.addLength() == 0 { // version 1 snapshot & 48k only
		s.version = 1
		if !s.sna && s.launcher.isCompressed() {
			log.WithField("version", s.version).Debug("decompressing snapshot")
			err = decompressZ80(rd, s.main[:49152])
		} else {
			log.WithField("version", s.version).Debug("reading snapshot")
			_, err = io.ReadFull(rd, s.main[:49152])
		}
		if err != nil {
			return err
		}

		var c byte
		if c, err = s.launcher.postSetup(rd); err != nil {
			return err
		}

		if s.launcher.isOtek() {

			var pageLayout [7]int
			for i := range pageLayout {
				pageLayout[i] = 99
			}
			pageLayout[0] = int(c & 7)

			switch pageLayout[0] {
			case 0:
				pageLayout[0] = 32768
				pageLayout[1] = 49152
				pageLayout[2] = 65536
				pageLayout[3] = 81920
				pageLayout[4] = 98304
				pageLayout[5] = 114688
			case 1:
				pageLayout[0] = 49152
				pageLayout[1] = 32768
				pageLayout[2] = 65536
				pageLayout[3] = 81920
				pageLayout[4] = 98304
				pageLayout[5] = 114688
			case 2:
				pageLayout[0] = 16384
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 65536
				pageLayout[4] = 81920
				pageLayout[5] = 98304
				pageLayout[6] = 114688
			case 3:
				pageLayout[0] = 65536
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 81920
				pageLayout[4] = 98304
				pageLayout[5] = 114688
			case 4:
				pageLayout[0] = 81920
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 65536
				pageLayout[4] = 98304
				pageLayout[5] = 114688
			case 5:
				pageLayout[0] = 0
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 65536
				pageLayout[4] = 81920
				pageLayout[5] = 98304
				pageLayout[6] = 114688
			case 6:
				pageLayout[0] = 98304
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 65536
				pageLayout[4] = 81920
				pageLayout[5] = 114688
			default:
				pageLayout[0] = 114688
				pageLayout[1] = 32768
				pageLayout[2] = 49152
				pageLayout[3] = 65536
				pageLayout[4] = 81920
				pageLayout[5] = 98304
			}

			if pageLayout[0] != 32768 {
				for i := 0; i < 16384; i++ {
					s.main[pageLayout[0]+i] = s.main[32768+i] //copy 0->?
				}
			}

			for i := 1; i < 7; i++ {
				if p := pageLayout[i]; p != 99 {
					log.WithFields(
						log.Fields{"address": p, "index": i}).Debug("reading page")
					if _, err := io.ReadFull(rd, s.main[p:p+16384]); err != nil {
						return err
					}
				}
			}
		}

	} else { // version 2 & 3
		if s.launcher.addLength() == 23 {
			s.version = 2
		} else {
			s.version = 3
		}
		log.WithField("version", s.version).Debug("reading snapshot")

		// Byte    Length  Description
		// -------------------------- -
		// 0       2       Length of compressed data(without this 3 - byte header)
		//                 If length = 0xffff, data is 16384 bytes longand not
		//                 compressed
		// 2       1       Page number of block
		//
		// for 48k snapshots the order is:
		//    0 48k ROM, 1, IF1/PLUSD/DISCIPLE ROM, 4 page 2, 5 page 0, 8 page 5,
		//    11 MF ROM only 4, 5 & 8 are valid for this usage, all others are
		//    just ignored
		// for 128k snapshots the order is:
		//    0 ROM, 1 ROM, 3 Page 0....10 page 7, 11 MF ROM.
		// all pages are saved and there is no end marker
		//

		var c byte
		for ; bankEnd > 0; bankEnd-- {
			if length, err = readUInt16(rd); err != nil {
				return err
			}

			if c, err = rd.ReadByte(); err != nil {
				return err
			}

			if int(c) >= len(s.bank) {
				return fmt.Errorf("corrupted snapshot data")
			}
			addr := s.bank[c]

			if addr != 99 {
				target := s.main[addr : addr+16384]
				if length == 65535 {
					_, err = io.ReadFull(rd, target)
				} else {
					err = decompressZ80(rd, target)
				}
				if err != nil {
					return err
				}
			}
		}
	}

	var hwm string
	if hwm, err = validateHardwareMode(
		s.launcher.hardwareMode(), s.version); err != nil {
		return err
	}

	if _, err := s.launcher.adjustStackPos(s.main, s.sna); err != nil {
		return err
	}

	size := "48k"
	if s.launcher.isOtek() {
		size = "128k"
	}

	log.WithFields(log.Fields{
		"size":          size,
		"h/w_mode":      hwm,
		"h/w_mode_byte": s.launcher.hardwareMode()}).Debug("snapshot read")

	return nil
}
