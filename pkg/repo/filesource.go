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

package repo

import (
	"bufio"
	"io"
	"os"
)

//
func NewFileSource(file string) (*FileSource, error) {
	if f, err := os.Open(file); err != nil {
		return nil, err
	} else {
		return &FileSource{file: f, reader: bufio.NewReader(f)}, nil
	}
}

//
type FileSource struct {
	file   *os.File
	reader io.Reader
}

//
func (fs *FileSource) Read(p []byte) (n int, err error) {
	return fs.reader.Read(p)
}

//
func (fs *FileSource) Close() error {
	return fs.file.Close()
}
