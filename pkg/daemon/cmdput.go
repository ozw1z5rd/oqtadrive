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
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/microdrive"
	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
	"github.com/xelalexv/oqtadrive/pkg/util"
)

/*
	The PUT command is used to send sections (header or record) to the daemon.

	variable length section, initially pending:

		arg 0:	drive number
		    1:	`0`
		    2:	`0` to go ahead, `1` for abort; conduit determines length
		        based on contents of initial bytes

		This requires highly reliable section data, and is used when recording
		data sent by IF1/QL.

	fixed length section:

		arg 0:	drive number
		    1:	section length high byte +1, i.e. never `0`
		    2:	section length low byte

		The last byte of the received section flags whether the section should
		be accepted (0) or rejected (>0). Values larger than 0 indicate the
		reason for rejection:

			1:	section too short
			2:  section too long

		This mode is used during drive shadowing, as data is not very reliable
		there, so variable length section cannot be used.
*/

//
// FIXME: - cleanup
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
func (c *command) putFixedLength(d *Daemon, drive int) (err error) {

	if !d.IsHardwareDrive(drive) {
		return fmt.Errorf("only use fixed length PUT during shadowing")
	}

	// received data won't have the preamble, we need to add this here
	data := make([]byte, int((c.arg(1)-1))*256+int(c.arg(2))+12)
	if err := d.conduit.receive(data[d.conduit.fillPreamble(data):]); err != nil {
		return err
	}

	code := data[len(data)-1] // rejection code
	if code != 0 {
		log.WithFields(
			log.Fields{"drive": drive, "code": code}).Debug("PUT rejected")
		d.mru.reset()
		return nil
	}

	data = data[:len(data)-1] // remove rejection code
	discard := true

	if len(data) < 200 {

		hd, err := microdrive.NewHeader(d.conduit.client, data, true)

		if err != nil { // FIXME replace corrupted header with generated one
			log.Warnf("error creating header: %v", err)
			hd.Emit(os.Stdout) // FIXME
		}

		if hd == nil {
			log.Warn("no header created")

		} else {
			log.WithField("sector", hd.Index()).Info("created header")
			d.mru.reset()
			if err = d.mru.setHeader(hd); err != nil {
				log.Errorf("error setting header: %v", err)
			} else {
				log.WithField("sector", hd.Index()).Info("set header")
				discard = false
			}
		}

	} else {

		rec, err := microdrive.NewRecord(d.conduit.client, data, true)

		if err != nil {
			log.Warnf("error creating record: %v", err)
			rec.Emit(os.Stdout) // FIXME

		}

		if rec == nil {
			log.Warn("no record created")

		} else {
			log.Info("created record")
			if err = d.mru.setRecord(rec); err != nil {
				log.Errorf("error setting record: %v", err)
			} else {
				log.Info("set record")
				discard = false
			}
		}
	}

	if discard {
		d.mru.reset()
		return nil
	}

	if d.mru.isNewSector() {

		sec, err := d.mru.createSector()
		if err != nil {
			log.Warnf("error creating sector: %v", err)
		}

		if cart := d.getCartridge(drive); cart != nil {

			logger := log.WithFields(log.Fields{
				"drive":  drive,
				"sector": sec.Index()})

			if p := cart.GetSectorAt(sec.Index()); p != nil {
				if hd := p.Header(); hd == nil || hd.ValidationError() != nil {
					p.SetHeader(sec.Header())
					shadowAnnotate(cart, p, "health.headers.bad")
					logger.Debug("PUT header amended")
				}
				if rec := p.Record(); rec == nil || rec.ValidationError() != nil {
					p.SetRecord(sec.Record())
					shadowAnnotate(cart, p, "health.records.bad")
					logger.Debug("PUT record amended")
				}

			} else {
				cart.SetSectorAt(sec.Index(), sec)
				shadowAnnotate(cart, sec, "")
				logger.Debug("PUT sector complete")
			}

		} else {
			return fmt.Errorf("error creating sector: no cartridge")
		}
	}

	return nil
}

//
func shadowAnnotate(cart base.Cartridge, sector base.Sector, ammended string) {

	if ammended == "" { // new sector added
		adjustShadowAnnotation(cart, "health.sectors", 1)
		bad := false
		if sector.Header().ValidationError() != nil {
			adjustShadowAnnotation(cart, "health.headers.bad", 1)
			bad = true
		}
		if sector.Record().ValidationError() != nil {
			adjustShadowAnnotation(cart, "health.records.bad", 1)
			bad = true
		}
		if bad {
			adjustShadowAnnotation(cart, "health.sectors.bad", 1)
		}

	} else {
		adjustShadowAnnotation(cart, ammended, -1)
		if sector.Header().ValidationError() == nil &&
			sector.Record().ValidationError() == nil {
			adjustShadowAnnotation(cart, "health.sectors.bad", -1)
		}
	}

	cart.SetModified(true)
}

//
func adjustShadowAnnotation(cart base.Cartridge, key string, val int) {
	cart.Annotate(key, getShadowAnnotation(cart, key).Int()+val)
}

//
func getShadowAnnotation(cart base.Cartridge, key string) *util.Annotation {
	if cart.HasAnnotation(key) {
		return cart.GetAnnotation(key)
	}
	return cart.Annotate(key, 0)
}
