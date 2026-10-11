# bhuppi

bhuppi is a parent atlas for everyday Unix jobs. You name the job (`files list`, `text awk fields`, `net ssh connect`). bhuppi prints the shape, then runs the real binary already on the machine (`ls`, `awk`, `ssh`).

It is not a coreutils rewrite, not a GNU-on-BSD translator, and not a shell. `bhuppi awk` is not a verb. `bhuppi explain awk` points at `text awk`.

```text
Wrap the real binary. Teach the shape. Never pretend GNU is BSD.
```

## Install

From this directory, on macOS or Ubuntu:

```sh
./install
bhuppi help
```

On Windows, from the same folder:

```bat
.\install.cmd
bhuppi help
```

`./install` uses a Go 1.22 or newer toolchain if one is already on `PATH`. It does not reinstall Go in that case. If Go is missing or older, it fetches the official Go 1.22.12 tarball from go.dev. Python is not required to build or run.

The binary is copied to a directory already on `PATH` (`/usr/local/bin` when that is the one available, with `sudo` if the directory is not writable). A bash/zsh hook is written under `~/.config/bhuppi` and sourced from your rc files so that `files go` and `folders go` can `cd` the current shell. The current terminal can run `bhuppi help` immediately. The `cd` hook is there in the next shell.

## Help is the product

`bhuppi`, `bhuppi help`, and `bhuppi --help` are the same overview. One level down, `bhuppi help files` and `bhuppi files --help` are the same page. Deeper pages (`bhuppi help text awk fields`) use that same renderer.

```text
--dry-run           print argv, do not exec
--os linux|darwin   render the other OS's examples (does not change exec)
--plain             no color / boxes
-h / --help         the same renderer as bhuppi help …
```

`bhuppi h` is help. There is no `bp` binary.

## Layout

| Path | Role |
| --- | --- |
| `cmd/bhuppi` | static binary |
| `internal/catalog/data.yaml` | embedded job tree (generated) |
| `internal/help` | help renderer |
| `internal/dispatch` | job → argv |
| `internal/execx` | PATH lookup, Windows maps, exec |
| `scripts/gen_catalog.py` | regenerates the catalog |
| `share/` | shell hook, PowerShell hook, `bhuppi.cmd` |
| `docs/tutorial.md` | Unix command → job tree |

Regenerate the catalog with `python3 scripts/gen_catalog.py`, then `go test ./...`.

## What it will not do

Missing binaries exit 2 with an install hint. Linux-only tools on macOS name the darwin page (`ss` → `lsof`) and the reverse. `nmap` is a docs page and does not scan. `rm` never gains `-rf` unless you typed that flag. GNU flags are not rewritten into BSD flags.
