# Cartridge Repository

You can store your collection of cartridges in a so-called *repo* on the daemon host. This is essentially just a designated folder where you place your cartridges. When set up, you can load cartridges from this repo with `repo://{path}` references, where `{path}` is the path to the desired cartridge, relative to the repo folder.

## Setup

1. Choose or create a root folder for the repo. Anywhere is fine as long as the user under which you run the daemon has access to this location.

2. Copy your files into the folder. Any supported cartridge format is fine. You can freely organize your files in sub-folders.

3. When starting the daemon with `oqtactl serve`, point it to the repo folder with the `--repo` or `-r` option, to make it aware of the repo. If you're running the daemon as a `systemd` service, you need to edit the unit file accordingly, and restart the service.

You can now use `oqtactl search` from anywhere on your network to search the files in the repo, and use a result when loading, e.g. `oqtactl load -i repo://a/b/pacman.z80`. Search & load is also supported in the web UI.

## Search Index
For quick search results, in particular incremental search in the web UI, the daemon automatically creates an index of the file names in the repo, and keeps track of any file changes (addition, removal, rename, move). Should you ever experience any problems with search, you can delete the index. It is located in the daemon's working directory, named `repo.index`. The daemon will recreate it upon restart.
