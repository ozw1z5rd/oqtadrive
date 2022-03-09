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

package format

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"

	log "github.com/sirupsen/logrus"
)

//
func NewCartReader(r io.ReadCloser, compressor string) (*CartReader, error) {

	log.WithField("compressor", compressor).Debug("cartridge reader requested")

	var ret *CartReader
	var err error

	switch compressor {

	case "gzip":
		fallthrough
	case "gz":
		ret, err = getGZipReader(r)

	case "zip":
		ret, err = getZipReader(r, false)

	case "7z":
		ret, err = getZipReader(r, true)

	case "":
		ret = &CartReader{r, "", "", ""}
	}

	if ret == nil {
		err = fmt.Errorf("unsupported compressor")
	}

	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"compressor": ret.compressor,
		"name":       ret.name,
		"type":       ret.typ}).Debug("cartridge reader created")

	return ret, nil
}

//
type CartReader struct {
	readCloser io.ReadCloser
	//
	name       string
	typ        string
	compressor string
}

//
func (r *CartReader) Read(p []byte) (n int, err error) {
	return r.readCloser.Read(p)
}

//
func (r *CartReader) Close() error {
	return r.readCloser.Close()
}

//
func (r *CartReader) Name() string {
	return r.name
}

//
func (r *CartReader) Type() string {
	return r.typ
}

//
func (r *CartReader) Compressor() string {
	return r.compressor
}

//
func getGZipReader(r io.ReadCloser) (*CartReader, error) {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	ret := &CartReader{readCloser: gzr}
	ret.name, ret.typ, _ = SplitNameTypeCompressor(gzr.Name)
	ret.compressor = "gzip"

	return ret, nil
}

//
func getZipReader(r io.ReadCloser, zip7 bool) (*CartReader, error) {

	var sponge bytes.Buffer
	size, err := io.Copy(&sponge, r)
	if err != nil {
		return nil, err
	}
	r.Close()

	ret := &CartReader{}

	if zip7 {
		zr, err := sevenzip.NewReader(bytes.NewReader(sponge.Bytes()), size)
		if err != nil {
			return nil, err
		}
		if len(zr.File) == 0 {
			return nil, fmt.Errorf("empty 7-zip archive")
		}
		if len(zr.File) > 1 {
			log.Warn("7-zip archive has more than one entry, using first")
		}

		ret.name, ret.typ, _ = SplitNameTypeCompressor(zr.File[0].Name)
		ret.compressor = "7z"
		ret.readCloser, err = zr.File[0].Open()

	} else {
		zr, err := zip.NewReader(bytes.NewReader(sponge.Bytes()), size)
		if err != nil {
			return nil, err
		}
		if len(zr.File) == 0 {
			return nil, fmt.Errorf("empty zip archive")
		}
		if len(zr.File) > 1 {
			log.Warn("zip archive has more than one entry, using first")
		}

		ret.name, ret.typ, _ = SplitNameTypeCompressor(zr.File[0].Name)
		ret.compressor = "zip"
		ret.readCloser, err = zr.File[0].Open()
	}

	if err != nil {
		return nil, err
	}

	return ret, nil
}

//
func SplitNameTypeCompressor(file string) (name, typ, compressor string) {

	_, n := filepath.Split(file)

	for {
		ext := filepath.Ext(n)
		if ext == "" {
			name = n
			break
		}

		n = strings.TrimSuffix(n, ext)
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))

		switch ext {

		case "mdr":
			fallthrough
		case "mdv":
			fallthrough
		case "z80":
			typ = ext

		case "gz":
			fallthrough
		case "gzip":
			fallthrough
		case "zip":
			fallthrough
		case "7z":
			compressor = ext
		}
	}

	return name, typ, compressor
}
