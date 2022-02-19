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

package util

//
func NewAnnotation(key string, value interface{}) *Annotation {
	return &Annotation{key: key, value: value}
}

//
type Annotation struct {
	key   string
	value interface{}
}

//
func (a *Annotation) IsBool() bool {
	_, ok := a.value.(bool)
	return ok
}

//
func (a *Annotation) Bool() bool {
	if v, ok := a.value.(bool); ok {
		return v
	}
	return false
}

//
func (a *Annotation) IsInt() bool {
	_, ok := a.value.(int)
	return ok
}

//
func (a *Annotation) Int() int {
	if v, ok := a.value.(int); ok {
		return v
	}
	return 0
}

//
func (a *Annotation) IsString() bool {
	_, ok := a.value.(string)
	return ok
}

//
func (a *Annotation) String() string {
	if v, ok := a.value.(string); ok {
		return v
	}
	return ""
}
