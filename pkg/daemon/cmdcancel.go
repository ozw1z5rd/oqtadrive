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

package daemon

import (
	log "github.com/sirupsen/logrus"
)

/*
	The CANCEL command is used to cancel a currently in-progress PUT.

		arg 0:	drive number
		    1:	cancel code
*/

// FIXME: maybe not necessary, use PUT with cancel code instead?
func (c *command) cancel(d *Daemon) error {

	drive, err := c.drive()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"drive": drive, "code": c.arg(1)}).Debug("CANCEL")
	d.mru.reset()

	return nil
}
