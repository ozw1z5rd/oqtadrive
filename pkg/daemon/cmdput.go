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
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/microdrive"
)

/*
	The PUT command is used to send sections (header or record) to the daemon.

	variable length section, pending:

		arg 0:	drive number
		    1:	`0`
		    2:	`0` to go ahead, `1` for abort; conduit determines length
		        based on contents of initial bytes

	fixed length section:

		arg 0:	drive number
		    1:	section length high byte +1, i.e. never `0`
		    2:	section length low byte

	The last byte of the received section flags whether the section should be
	accepted (0) or rejected (>0). Values larger than 0 indicate the reason for
	rejection.
*/

//
func (c *command) put(d *Daemon) error {

	drive, err := c.drive()
	if err != nil {
		return err
	}

	if c.arg(1) == 0 {
		return c.putVariableLength(d, drive)
	}
	return c.putFixedLength(d, drive)
}

//
func (c *command) putVariableLength(d *Daemon, drive int) error {

	if c.arg(2) != 0 { // ignore canceled PUT
		log.WithFields(
			log.Fields{"drive": drive, "code": c.arg(2)}).Debug("PUT aborted")
		return nil
	}

	if d.IsHardwareDrive(drive) {
		return fmt.Errorf("must not use variable length PUT during shadowing")
	}

	data, err := d.conduit.receiveBlock()
	if err != nil {
		return err
	}

	if len(data) < 200 {
		if hd, err := microdrive.NewHeader(d.conduit.client, data, true); err != nil {
			return fmt.Errorf("error creating header: %v", err)
		} else if err = d.mru.setHeader(hd); err != nil {
			return err
		}

	} else {
		if rec, err := microdrive.NewRecord(d.conduit.client, data, true); err != nil {
			return fmt.Errorf("error creating record: %v", err)
		} else if err = d.mru.setRecord(rec); err != nil {
			return err
		}

		if d.mru.isRecordUpdate() {
			defer d.mru.reset()
			if cart := d.getCartridge(drive); cart != nil {
				cart.SetModified(true)
				log.WithFields(log.Fields{
					"drive":  drive,
					"sector": d.mru.sector.Index(),
				}).Debug("PUT record")
			} else {
				return fmt.Errorf("error updating record: no cartridge")
			}
		}
	}

	if d.mru.isNewSector() {
		sec, err := d.mru.createSector()
		if err != nil {
			return fmt.Errorf("error creating sector: %v", err)
		}

		if cart := d.getCartridge(drive); cart != nil {
			cart.SetNextSector(sec)
			log.WithFields(log.Fields{
				"drive":  drive,
				"sector": sec.Index(),
			}).Debug("PUT sector complete")
		} else {
			return fmt.Errorf("error creating sector: no cartridge")
		}
	}

	return nil
}

//
func (c *command) putFixedLength(d *Daemon, drive int) error {

	if !d.IsHardwareDrive(drive) {
		return fmt.Errorf("only use fixed length PUT during shadowing")
	}

	ln := int((c.arg(1)-1))*256 + int(c.arg(2))
	data := make([]byte, ln)

	if err := d.conduit.receive(data); err != nil {
		return err
	}

	//
	code := data[ln-1]
	if code != 0 {
		log.WithFields(
			log.Fields{"drive": drive, "code": code}).Debug("PUT rejected")
		d.mru.reset()
		return nil
	}

	ln--
	data = data[:ln]

	if ln < 200 {
		hd, err := microdrive.NewHeader(d.conduit.client, data, true)
		if err != nil {
			log.Warnf("error creating header: %v", err)
		}
		d.mru.reset()
		if err = d.mru.setHeader(hd); err != nil {
			log.Errorf("error setting header: %v", err)
			d.mru.reset()
			return nil
		}

	} else {
		rec, err := microdrive.NewRecord(d.conduit.client, data, true)
		if err != nil {
			log.Warnf("error creating record: %v", err)
		}
		if err = d.mru.setRecord(rec); err != nil {
			log.Errorf("error setting record: %v", err)
			d.mru.reset()
			return nil
		}
	}

	if d.mru.isNewSector() {
		sec, err := d.mru.createSector()
		if err != nil {
			log.Warnf("error creating sector: %v", err)
		}

		if cart := d.getCartridge(drive); cart != nil {
			cart.SetSectorAt(sec.Index(), sec)
			log.WithFields(log.Fields{
				"drive":  drive,
				"sector": sec.Index(),
			}).Debug("PUT sector complete")
		} else {
			return fmt.Errorf("error creating sector: no cartridge")
		}
	}

	return nil
}
