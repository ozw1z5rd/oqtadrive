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

package base

import (
	"fmt"
	"io"
)

// This is the length of a file block contained in a record, i.e. the effective
// record payload. It is the same for Spectrum & QL. A file block starts at
// index 0 within the record data. In certain situations, e.g. Spectrum with old
// ROM, the data section of a record may be longer than the file block length.
const FileBlockLength = 512

//
type FileImpl interface {

	// length of a file header applicable to the record implementation; the file
	// header is only contained at the start of the first record of a file
	FileHeaderLength() int
}

//
func NewFile(name string, size int, records []Record, impl FileImpl) *File {
	return &File{
		fInfo:   *NewFileInfo(name, size),
		fImpl:   impl,
		records: records,
	}
}

// for keeping embedded anonymous fields package private
type fInfo = FileInfo
type fImpl = FileImpl

//
type File struct {
	//
	fInfo
	fImpl
	//
	records []Record
	readPos int
}

//
func (f *File) Bytes() ([]byte, error) {

	b := make([]byte, f.Size())

	n, err := f.Read(b)
	if err != nil {
		return nil, err
	}

	return b[:n], nil
}

//
func (f *File) Read(p []byte) (int, error) {

	if len(p) == 0 {
		return 0, nil
	}

	skip := 0
	if f.readPos == 0 {
		skip = f.FileHeaderLength()
	}

	rIx, bIx, err := f.advanceReadPos(skip)
	if err != nil {
		return 0, err
	}

	read := 0

	for read < len(p) {

		if r := f.records[rIx]; r != nil {

			bEnd := FileBlockLength
			if rIx == len(f.records)-1 {
				if e := f.Size() % FileBlockLength; e > 0 {
					bEnd = e
				}
			}

			n := copy(p[read:], r.Data()[bIx:bEnd])
			read += n

			if rIx, bIx, err = f.advanceReadPos(n); err != nil {
				return read, err
			}

		} else {
			return read, fmt.Errorf("missing record at index %d", rIx)
		}
	}

	return read, nil
}

//
func (f *File) advanceReadPos(n int) (recIx, blockIx int, err error) {

	f.readPos += n
	recIx = f.readPos / FileBlockLength
	blockIx = f.readPos % FileBlockLength
	err = nil

	if f.readPos >= f.Size() || recIx >= len(f.records) {
		err = io.EOF
	}

	return
}
