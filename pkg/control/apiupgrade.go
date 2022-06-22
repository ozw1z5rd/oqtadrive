//go:build linux || darwin

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

package control

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
)

//
func (a *api) upgrade(w http.ResponseWriter, req *http.Request) {

	pwd, err := os.Getwd()
	if handleError(err, http.StatusInternalServerError, w) {
		return
	}

	cmd := exec.Command("bash", "-c", "make can_upgrade")
	cmd.Dir = pwd
	if err = cmd.Run(); err != nil {
		handleError(fmt.Errorf("upgrade via API not supported: %v", err),
			http.StatusServiceUnavailable, w)
		return
	}

	// FIXME: use this over ForkExec? ForkExec has the advantage of timely log
	//        output.
	//
	// cmd = exec.Command("bash", "-c", "source /home/pi/.bashrc; make upgrade")
	// cmd.Dir = pwd
	// if err = cmd.Start(); err != nil {
	// 	handleError(fmt.Errorf("upgrade via API not supported"),
	// 		http.StatusInternalServerError, w)
	// 	return
	// }

	bash, err := exec.LookPath("bash")
	if handleError(err, http.StatusInternalServerError, w) {
		return
	}

	bash, err = filepath.Abs(bash)
	if handleError(err, http.StatusInternalServerError, w) {
		return
	}

	env := os.Environ()
	if buildURL := getArg(req, "build_url"); buildURL != "" {
		env = append(env, fmt.Sprintf("%s=%s", "BUILD_URL", buildURL))
	}

	// FIXME: do we need `bash` in argv?
	_, err = syscall.ForkExec(bash, []string{"bash", "-c", "make upgrade"},
		&syscall.ProcAttr{
			Env: env,
			Dir: pwd,
			Sys: &syscall.SysProcAttr{
				Setsid: true,
			},
			Files: []uintptr{0, 1, 2},
		})

	if handleError(err, http.StatusServiceUnavailable, w) {
		return
	}

	log.WithField("workdir", pwd).Info("upgrade process started")

	sendReply([]byte("The upgrade was triggered. The daemon will go down soon "+
		"and restart when the upgrade is done. Note that there won't be a "+
		"notification about the completion of the upgrade."),
		http.StatusOK, w)
}
