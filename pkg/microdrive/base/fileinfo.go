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
	"github.com/xelalexv/oqtadrive/pkg/util"
)

//
func NewFileInfo(name string, size int) *FileInfo {
	return &FileInfo{name: name, size: size}
}

//
type FileInfo struct {
	name string
	size int
	util.Annotations
}

//
func (f *FileInfo) Name() string {
	return f.name
}

//
func (f *FileInfo) Size() int {
	return f.size
}
