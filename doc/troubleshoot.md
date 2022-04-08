# Troubleshooting Guide

## *Use the Log, Luke!*
Whenever you hit a problem, first thing to do is to check the daemon log. For this, run the daemon preferably with `debug` log level, e.g.:

```bash
LOG_LEVEL=debug oqtactl serve -d /dev/ttyUSB0 [other options]
```

If you're running the daemon as a `systemd` service, you need to edit the unit file accordingly, and restart the service. For many problems, the daemon log should already give you good hints about what's going on. It's also important when reporting an issue over at *GitHub*.

## Check the Online Help
The `oqtactl` binary provides online help for all actions. Just run `oqtactl --help` to get a list of the available actions, and `oqtactl {action} --help` for finding out more about a particular action. Maybe there are additional options that you weren't aware of, and that can help you work around a problem.

## Make Sure *OqtaDrive* Correctly Recognizes Your Machine
When the *OqtaDrive* adapter starts up, it auto-detects what it's connected to, i.e. *Interface 1* or *QL*, and configures itself accordingly. This may fail in certain situations, so it's one of the first things to check. The daemon log and also the web UI show what machine has been detected.

If this is incorrect, re-sync the adapter, either via the web UI or `oqtactl resync`. The latter method also allows you to force a particular machine with the `--client` option. If you're always using the adapter with only one type of machine, you can also hard-code that in the adapter config. Check the config section at the top of `oqtadrive.ino`, and look for the `FORCE_IF1` and `FORCE_QL` settings. You can also force a particular machine with the `--client` option when starting the daemon. 

*Tip*: When using the adapter with a *Spectrum* + *Interface 1*, make sure to power up the adapter after or together with the *Spectrum*, but not before. Otherwise it will auto-detect a *QL*.

## Topics

### I cannot load this cartridge, keep getting `cartridge corrupted`
If loading a cartridge fails due to cartridge corruption (usually caused by incorrect check sums), try the `--repair` option of the `load` command. With this, *OqtaDrive* will try to repair the cartridge when loading.

### I see unstable behavior, keep getting `Microdrive not found`
Check the wiring that connects the adapter to the *Microdrive* port. Use ribbon cable, rather than separate wires, and keep the cable length below 5cm.

### I see drives at the wrong number
If you're accessing let's say drive 2, but get the content of drive 3, then the *adapter offset* may be wrong. This offset is the number of actual hardware drives present in the drive chain, upstream of the adapter. For the *QL*, this offset is by default auto-detected. When connecting the adapter for example to the external *Microdrive* port of a stock *QL*, it should be two. Auto-detect however may fail. For *Interface 1*, auto-detect is not possible and defaults to `0`. You can explicitly set a fixed offset in the config section at the top of `oqtadrive.ino`, settings `DRIVE_OFFSET_QL` and `DRIVE_OFFSET_IF1`.

### I reset my *Spectrum*/*QL*, but the adapter LED keeps flashing
This usually happens when resetting while a drive is active. The reason is that in the case of a reset on the *Spectrum* or *QL* side, the drive is never 'officially' deactivated by the host machine, so *OqtaDrive* keeps spinning the drive. You need to reset the adapter as well, either via web UI or `oqtactl resync --reset`, or quite simply via the *Arduino*'s own reset button. If you've installed the adapter inside the case of a *Spectrum*, *Interface 1*, or *QL*, consider hooking up the reset line of the *Nano* with the machine's reset line.

### I don't get any meaningful results from *repo* search
First, make sure you're using the right search term. The web UI has a help tool tip about this. Repo search may actually be doing what you asked. If that doesn't help, try removing the search index. Also note that it is not recommended to place the repo & search index onto a volume with a *FAT*-type file system. More details [here](repo.md).
