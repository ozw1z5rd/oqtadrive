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

package repo

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
	log "github.com/sirupsen/logrus"
)

//
type SearchResult struct {
	Hits     []string `json:"hits"`
	Total    uint64   `json:"total"`
	Complete bool     `json:"complete"`
}

//
func (i *Index) Search(term string, max int) (*SearchResult, error) {

	term = strings.TrimSpace(term)
	if term == "" {
		return nil, fmt.Errorf("no search term")
	}

	log.Debugf("searching for '%s'", term)
	query := bleve.NewQueryStringQuery(term)
	search := bleve.NewSearchRequestOptions(query, max+1, 0, false)
	res, err := i.index.Search(search)
	if err != nil {
		return nil, err
	}

	ret := &SearchResult{
		Hits:     make([]string, len(res.Hits)),
		Total:    res.Total,
		Complete: true}

	for ix, h := range res.Hits {
		ret.Hits[ix] = h.ID
	}

	if len(ret.Hits) > max {
		ret.Hits = ret.Hits[:max]
		ret.Complete = false
	}

	return ret, nil
}
