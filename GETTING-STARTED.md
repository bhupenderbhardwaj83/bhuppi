# Getting started

## macOS or Ubuntu

```sh
cd bhuppi
./install
bhuppi help
```

If Go 1.22 or newer is already installed, `./install` uses it and does not download another toolchain. Check that decision without building:

```sh
./install.sh --go-decision
```

`use` means the `go` on `PATH` is new enough. `fetch` means the installer would download Go 1.22.12. `--go-decision` never downloads.

Then:

```sh
bhuppi help files
bhuppi files list .
bhuppi --dry-run files list .
bhuppi doctor
```

`bhuppi files go src` changes directory only after a new shell has sourced the hook (`~/.config/bhuppi/zsh.sh` or `bash.sh`). Until then, `--dry-run` prints `cd src` and a live `files go` tells you the hook is required. A child process cannot `cd` its parent.

## Windows

```bat
.\install.cmd
bhuppi help
```

`install.cmd` runs `install.ps1`. The real binary is `%LOCALAPPDATA%\bhuppi\lib\bhuppi.exe`. The thing on `PATH` is `bhuppi.cmd`, so `folders go` can `cd` the cmd window. Do not put `bhuppi.exe` in that same directory: Windows tries `.EXE` before `.CMD`.

If Git for Windows `usr\bin` is missing, the installer runs `winget install Git.Git` once. When those Unix tools are on `PATH`, bhuppi keeps them (`ls` stays `ls`). Otherwise it maps a short list onto `cmd` (`ls` → `dir`, `cat` → `type`, `grep` → `findstr`). Windows `find.exe` and `sort.exe` are never treated as Unix `find` or `sort`.

## Read the tree

```sh
bhuppi help                  # overview
bhuppi help text             # the group
bhuppi help text awk         # language-mode, five children
bhuppi help text awk fields  # one feature
bhuppi explain awk           # A–Z index line
```

`bhuppi text awk fields --help` is the same page as `bhuppi help text awk fields`.

A longer map from ordinary Unix onto this tree is in [docs/tutorial.md](docs/tutorial.md).
