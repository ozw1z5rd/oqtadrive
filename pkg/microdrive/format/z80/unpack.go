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
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

//
func (s *snapshot) unpack(in io.Reader) error {

	rd := bufio.NewReader(in)

	if err := s.launcher.setup(rd); err != nil {
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

	var err error

	if s.launcher.addLength() == 0 { // version 1 snapshot & 48k only
		log.Debug("snapshot version: v1")
		s.version = 1
		if s.compressed {
			err = decompressZ80(rd, s.main)
		} else {
			_, err = io.ReadFull(rd, s.main)
		}
		if err != nil {
			return err
		}

	} else { // version 2 & 3
		if s.launcher.addLength() == 23 {
			log.Debug("snapshot version: v2")
			s.version = 2
		} else {
			log.Debug("snapshot version: v3")
			s.version = 3
		}

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
		if s.launcher.isOtek() {
			s.bank[3] = 32768   // page 0
			s.bank[4] = 49152   // page 1
			s.bank[5] = 16384   // page 2
			s.bank[6] = 65536   // page 3
			s.bank[7] = 81920   // page 4
			s.bank[8] = 0       // page 5
			s.bank[9] = 98304   // page 6
			s.bank[10] = 114688 // page 7
			s.bankEnd = 10
		} else {
			s.bank[4] = 16384 // page 2
			s.bank[5] = 32768 // page 0
			s.bank[8] = 0     // page 5
			s.bankEnd = 8
		}

		var c byte
		for c = 0; c != s.bankEnd; {
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

	rand := s.launcher.randomize()

	if s.launcher.isOtek() {
		log.Debug("snapshot size: 128k")
		s.code = make([]byte, len(mdrBl128k))
		copy(s.code, mdrBl128k)
		s.code[ix128kBrd] = s.launcher.borderColor()
		s.code[ix128kPap] = s.launcher.borderColor()
		s.code[ix128kUsr] = byte(rand)
		s.code[ix128kUsr+1] = byte(rand >> 8)
	} else {
		log.Debug("snapshot size: 48k")
		s.code = make([]byte, len(mdrBl48k))
		copy(s.code, mdrBl48k)
		s.code[ix48kBrd] = s.launcher.borderColor()
		s.code[ix48kPap] = s.launcher.borderColor()
		s.code[ix48kUsr] = byte(rand)
		s.code[ix48kUsr+1] = byte(rand >> 8)
	}

	return nil
}
