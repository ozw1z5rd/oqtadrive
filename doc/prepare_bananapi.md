## Preparing a *BananaPi M2 Zero*

- [Download the *Armbian* image](http://armbian.hosthatch.com/archive/bananapim2zero/archive/Armbian_21.08.1_Bananapim2zero_focal_current_5.10.60.img.xz). This is the last image that was officially released for the *BananaPi M2 Zero*, and dates back to August 2021. It still works, but if you want something more recent and are experienced enough, you can also [build the image yourself](https://github.com/armbian/build). The *Armbian* build system is quite solid, and even offers a *dockerized* build.

- Use an image writer (e.g. *gnome-disk-utility* on *Ubuntu*) to write the image to a suitable micro SD card.

- *Optional (but recommended)*: Use a partition editor such as *gparted* to increase the `root` partition to full size.

- For letting the *Pi* access your wireless network, copy `/boot/armbian_first_run.txt.template` to `/boot/armbian_first_run.txt` on the SD card. Make these changes:

    ```
    FR_general_delete_this_file_after_completion=1
    FR_net_change_defaults=1
    FR_net_ethernet_enabled=0
    FR_net_wifi_enabled=1
    FR_net_wifi_ssid='<your SSID>'
    FR_net_wifi_key='<your password>'
    FR_net_wifi_countrycode='<country code, upper case>'
    ```

    You may want to create a backup copy of this file when done, since it will get deleted after the first boot.

- To enable the serial port, edit `/boot/armbianEnv.txt` and add this line:

    ```
    overlays=uart3
    ```

    If an `overlays=...` line already exists, just add `uart3` to its end, separated with a space.

- For accessing the board upon first boot, you have three options:

    + connect monitor and keyboard (requires USB OTG adapter)
    + connect via serial console (haven't tried this myself)
    + connect as `root` via `ssh`, however only login with ssh key is allowed, so you have to create file `/root/.ssh/authorized_keys` with your public ssh key in it

- Place the SD card in the *Pi*, pick your method of connecting, and boot it up. If you want to connect via `ssh`, check your wireless router to find out which IP address the Pi received, and log in, e.g. `ssh root@192.168.1.12`. The *Armbian* configurator will guide you through a config session for setting up `root` password and a standard user, and a few other basic settings. Once done, you're in a shell session as `root`.

- By default, the new user created in the previous step is already added to the `sudo` group, but still requires the password to be entered. For correct functioning of the *OqtaDrive* installation, we need to enable password-less `sudo` for this user. Invoke `visudo` and add the following line to the end, replacing `{user}` with the name of your user:

    ```
    {user} ALL=(ALL) NOPASSWD:ALL
    ```

- If you want to connect with `ssh` using the standard user later on, you have to create `/home/{user}/.ssh/authorized_keys` with your public ssh key in it.

- Reboot: `sudo reboot`

## Running the Installer
This works the same as given in the main install guide. The only difference is that in addition to the `PORT` environment variable, you also have to set `BAUD_RATE` to `500000` and `RESET_PIN` to `16` (`68` if you're using an old standalone PCB), so running a full install would look like this:

```
PORT=/dev/ttyS3 BAUD_RATE=500000 RESET_PIN=16 make install patch_avrdude flash service_on
```

### Background
- The baud rate for the serial connection between daemon and adapter needs to be changed since the *BananaPi M2 Zero* does not support the standard 1 Mbps. The fastest speed that both the *Pi* and the *Arduino* support is 500 kbps. Despite not being ideal, this speed seems to be sufficient for normal operation of *OqtaDrive*, but has not been extensively tested yet. Feedback is welcome.
- The *BananaPi M2 Zero* uses a different GPIO chip and therefore different pin line numbering.
