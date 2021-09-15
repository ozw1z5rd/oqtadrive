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

package control

import (
	"fmt"
	"net/http"
	"strings"
)

//
func (a *api) search(w http.ResponseWriter, req *http.Request) {

	if a.index == nil {
		handleError(fmt.Errorf("search index not available"),
			http.StatusServiceUnavailable, w)
		return
	}

	items, err := getIntArg(req, "items", 100)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	res, err := a.index.Search(getArg(req, "term"), items)
	if handleError(err, http.StatusUnprocessableEntity, w) {
		return
	}

	if wantsJSON(req) {
		sendJSONReply(res, http.StatusOK, w)

	} else {
		var sb strings.Builder
		for _, r := range res.Hits {
			sb.WriteString(fmt.Sprintf("%s\n", r))
		}
		sb.WriteString(fmt.Sprintf("\ntotal hits: %d\n", res.Total))
		sendReply([]byte(sb.String()), http.StatusOK, w)
	}
}
