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

package daemon

import (
	log "github.com/sirupsen/logrus"
)

/*
	The MAP command is used to set/query hardware drive mapping & shadowing
	settings.

		arg 0:	number of first h/w drive
		    1:	number of last h/w drive
		    2:	flags:
		    		0: group locked
		    		1: shadowing
		    		7: if 1 while making settings, ignore args 0 & 1 and only
		    		   act on the flags, if 0 only act on the drive numbers
*/

//
func (c *command) driveMap(d *Daemon) error {

	d.conduit.hwGroupStart = int(c.arg(0))
	d.conduit.hwGroupEnd = int(c.arg(1))
	d.conduit.hwGroupLocked = c.arg(2)&MaskHWGroupLocked != 0
	d.conduit.hwShadowing = c.arg(2)&MaskHWShadowing != 0

	log.WithFields(log.Fields{
		"start":     c.arg(0),
		"end":       c.arg(1),
		"locked":    d.conduit.hwGroupLocked,
		"shadowing": d.conduit.hwShadowing}).Info("MAP")

	return nil
}
