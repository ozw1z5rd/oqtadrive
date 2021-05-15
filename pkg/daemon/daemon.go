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
	"context"
	"fmt"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/microdrive"
	"github.com/xelalexv/oqtadrive/pkg/microdrive/base"
)

//
const DriveCount = 8

// the daemon that manages communication with the Interface 1/QL
type Daemon struct {
	//
	cartridges []atomic.Value
	conduit    *conduit
	port       string
	synced     bool
	//
	mru        *mru
	debugStart time.Time
}

//
func NewDaemon(port string) *Daemon {
	return &Daemon{
		cartridges: make([]atomic.Value, DriveCount),
		port:       port,
		mru:        &mru{},
	}
}

//
func (d *Daemon) Serve() error {
	return d.listen()
}

//
func (d *Daemon) listen() error {

	if err := d.ResetConduit(); err != nil {
		return err
	}

	d.loadBlankCartridges()

	var cmd *command
	var err error

	for ; ; cmd = nil {
		if d.synced {
			if cmd, err = d.conduit.receiveCommand(); err != nil {
				log.Errorf("error receiving command: %v", err)
				d.synced = false
			}

		} else {
			if err = d.conduit.syncOnHello(); err != nil {
				log.Errorf("error syncing with adapter: %v", err)
			} else {
				d.synced = true
				for ix := 1; ix <= DriveCount; ix++ {
					if cart := d.getCartridge(ix); cart != nil {
						cart.Unlock()
					}
				}
			}
		}

		if err != nil {
			d.mru.reset()
			if err := d.ResetConduit(); err != nil {
				return err
			}

		} else if cmd != nil {
			if err = cmd.dispatch(d); err != nil {
				log.Errorf("error dispatching command: %v", err)
				d.synced = false
			}
		}
	}
}

//
func (d *Daemon) ResetConduit() error {

	logger := log.WithField("port", d.port)
	d.synced = false

	if d.conduit != nil {
		logger.Info("closing serial port")
		if err := d.conduit.close(); err != nil {
			log.Errorf("error closing serial port: %v", err)
		}
		d.conduit = nil
	}

	logger.Info("opening serial port")
	maxBackoff := 15 * time.Second
	quiet := false

	for backoff := time.Second; ; {
		if con, err := newConduit(d.port); err != nil {
			if !quiet {
				logger.Warnf("cannot open serial port: %v", err)
			}

			if backoff < maxBackoff {
				backoff *= 2
			} else if !quiet {
				logger.Warn(
					"repeatedly failed to open serial port, will keep trying but stop logging about it")
				quiet = true
			}
			time.Sleep(backoff)

		} else {
			logger.Info("serial port opened")
			d.conduit = con
			return nil
		}
	}
}

//
func (d *Daemon) loadBlankCartridges() {
	for ix := 1; ix <= len(d.cartridges); ix++ {
		if cart, err := microdrive.NewCartridge(d.conduit.client); err == nil {
			d.SetCartridge(ix, cart, true)
		}
	}
}

//
func (d *Daemon) UnloadCartridge(ix int, force bool) error {
	if d.conduit == nil {
		return fmt.Errorf("nothing to unload")
	}
	cart, err := microdrive.NewCartridge(d.conduit.client)
	if err != nil {
		return err
	}
	return d.SetCartridge(ix, cart, force)
}

// SetCartridge sets the cartridge at slot ix (1-based).
func (d *Daemon) SetCartridge(ix int, c base.Cartridge, force bool) error {

	if present, ok := d.GetCartridge(ix); !ok {
		return fmt.Errorf("could not lock present cartridge")

	} else if !force && present != nil && present.IsModified() {
		present.Unlock()
		return fmt.Errorf("present cartridge is modified")
	}

	d.setCartridge(ix, c)
	return nil
}

//
func (d *Daemon) setCartridge(ix int, c base.Cartridge) {
	if 0 < ix && ix <= len(d.cartridges) {
		d.cartridges[ix-1].Store(&c)
	}
}

// GetCartridge gets the cartridge at slot ix (1-based)
func (d *Daemon) GetCartridge(ix int) (base.Cartridge, bool) {

	cart := d.getCartridge(ix)

	if cart != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if cart.Lock(ctx) {
			return cart, true
		} else {
			return nil, false
		}
	}

	return nil, true
}

//
func (d *Daemon) getCartridge(ix int) base.Cartridge {
	if 0 < ix && ix <= len(d.cartridges) {
		if cart := d.cartridges[ix-1].Load(); cart != nil {
			return *(cart.(*base.Cartridge))
		}
	}
	return nil
}
