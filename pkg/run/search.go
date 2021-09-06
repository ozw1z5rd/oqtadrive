/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2021, Alexander Vollschwitz

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

package run

import (
	"fmt"
	"io"
	"net/url"
	"os"
)

//
func NewSearch() *Search {

	s := &Search{}
	s.Runner = *NewRunner(
		"search [-a|--address {address}] -t|--term {search term} [-i|--items {max results}]",
		"search for cartridges in daemon repo",
		`
Use the search command to find cartridges in the daemon's repository, if enabled.`,
		"", runnerHelpEpilogue, s.Run)

	s.AddBaseSettings()
	s.AddSetting(&s.Term, "term", "t", "", nil,
		"search term; used to search through the cartridge file names", true)
	s.AddSetting(&s.Items, "items", "i", "", 100,
		"max number of search results to return", false)

	return s
}

//
type Search struct {
	Runner
	//
	Term  string
	Items int
}

//
func (s *Search) Run() error {

	s.ParseSettings()

	resp, err := s.apiCall("GET",
		fmt.Sprintf("/search?items=%d&term=%s", s.Items, url.QueryEscape(s.Term)),
		false, nil)
	if err != nil {
		return err
	}
	defer resp.Close()

	fmt.Println()
	if _, err := io.Copy(os.Stdout, resp); err != nil {
		return err
	}

	return nil
}
