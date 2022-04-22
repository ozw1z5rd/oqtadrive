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
	"io"
	"net/http"
)

//
func NewHTTPSource(url string) (*HTTPSource, error) {
	if resp, err := http.Get(url); err != nil {
		return nil, err
	} else {
		return &HTTPSource{
			url:      url,
			response: resp,
			reader:   io.LimitReader(resp.Body, 1048576)}, nil
	}
}

//
type HTTPSource struct {
	url      string
	response *http.Response
	reader   io.Reader
}

//
func (hs *HTTPSource) Read(p []byte) (n int, err error) {
	return hs.reader.Read(p)
}

//
func (hs *HTTPSource) Close() error {
	return hs.response.Body.Close()
}
