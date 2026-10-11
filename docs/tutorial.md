# Unix, on the job tree

bhuppi teaches a job, then execs the binary you already have. The same page is `bhuppi help …` and `… --help`. Nothing here is a second implementation of `ls` or `awk`.

```text
Wrap the real binary. Teach the shape. Never pretend GNU is BSD.
```

Shape legend, on every page:

```text
UPPERCASE = you supply it · [brackets] = optional · a|b = pick one
```

## Four depths

**Overview** — `bhuppi`, `bhuppi help`, `bhuppi --help`.

The identity box is only here. Fifteen groups. Five are marked ✨ because they are the first place to look: `files`, `folders`, `text`, `proc`, `net`.

**Group** — `bhuppi help files` is the same page as `bhuppi files --help`.

```text
bhuppi files — everyday files
WHAT: …
WHEN: …
RULE: …
```

Then the child verbs, each with a shape, one example, and a short description.

**Language-mode** — `bhuppi help text awk`. Same contract. At most five children, listed the way a group lists commands. The page still runs real `awk`.

**Feature** — `bhuppi help text awk fields`.

```text
bhuppi text awk fields — print fields
WHAT / WHEN / RULE
shape | e.g. bhuppi text awk fields 1,3 a.log
runs: awk '{print $1,$3}' a.log
NOT THIS
more:
```

`--dry-run` prints that same argv and does not exec:

```sh
bhuppi --dry-run text awk fields 1,3 a.log
# awk '{print $1,$3}' a.log
```

## First files and folders

| You would type | Job | Real argv |
| --- | --- | --- |
| `ls` | `bhuppi files list [PATH]` | `ls` |
| `cat FILE` | `bhuppi files show FILE` | `cat` |
| `pwd` | `bhuppi files where` | `pwd` |
| `cd PATH` | `bhuppi files go PATH` | `cd` (shell hook) |
| `mkdir DIR` | `bhuppi files make DIR` | `mkdir` |
| `touch FILE` | `bhuppi files make FILE --empty` | `touch` |
| `cp SRC DST` | `bhuppi files copy SRC DST` | `cp` |
| `mv SRC DST` | `bhuppi files move SRC DST` | `mv` |
| `rm PATH` | `bhuppi files remove PATH` | `rm`, or `rmdir` if the directory is empty |
| `find` / `fd` | `bhuppi files find NAME [PATH]` | `fd --glob` when `fd` is installed, else `find` |

`folders` repeats the directory jobs: `go`, `make`, `remove`, `list`, `copy`, `move`, `find`. `folders make` is only `mkdir` (`--empty` belongs to `files make`). `folders remove` is always `rmdir`. `files remove` runs `rm` and adds `-r` or `-f` only when you typed them.

Quote globs so the shell does not expand them first:

```sh
bhuppi files find '*.go' .
```

## Text, languages, data

| Unix | Job |
| --- | --- |
| `awk '{print $1,$3}' a.log` | `bhuppi text awk fields 1,3 a.log` |
| `awk` program | `bhuppi text awk filter`, `begin-end`, `oneline`, `script` |
| `sed` substitute | `bhuppi text sed replace` |
| `sed` delete | `bhuppi text sed delete` |
| `sed -i` | `bhuppi text sed inplace` |
| `bash` | `bhuppi lang bash run`, `check`, `flags`, `builtin` |
| `jq` | `bhuppi data jq get`, `filter`, `keys`, `slurp` |
| `yq` | `bhuppi data yq` (same four) |
| `openssl` | `bhuppi data openssl key`, `csr`, `cert`, `verify`, `handshake` |

`sed inplace` is honest about the host: Linux `sed -i`, macOS `sed -i ''`. `--os darwin` changes the help example. It does not change a live exec, and it does not rewrite flags you typed yourself.

`lang bash builtin` teaches `cd`, `export`, `history`, `type`, `test`, `alias`, `source`, `eval`, `jobs`, `bg`, `fg`. Those stay in the shell. Only `files go` / `folders go` is hooked.

## Processes, net, packages, disk, services

| Unix | Job |
| --- | --- |
| `ps` | `bhuppi proc list` |
| `lsof` | `bhuppi proc using` |
| `ssh` | `bhuppi net ssh connect` |
| `ssh-keygen` | `bhuppi net ssh key` |
| `scp` | `bhuppi net ssh copy` |
| `ss` / `lsof` | `bhuppi net ports` |
| `tar -czf` | `bhuppi pack tar compress logs.tgz logs` |
| `apt` / `brew` search | `bhuppi host pkg search` |
| `systemctl` | `bhuppi svc systemctl status`, `start`, `stop`, `enable` |
| `journalctl -u` | `bhuppi svc systemctl logs` |
| `diskutil` | `bhuppi disk diskutil list` (darwin) |
| `launchctl` | `bhuppi svc launchctl list` (darwin) |

`net filter list` only names `iptables`, `nft`, `ufw`, or `pfctl`. Extra flags are refused. `disk fdisk` is `fdisk -l` only. `tcpdump` accepts an interface and an optional count.

## Linux, macOS, Windows

`--os linux` or `--os darwin` redraws examples. The process you exec is still the one on this machine.

When a Linux-only binary is missing on macOS, bhuppi exits 2, names it, and points at the darwin page (`ss` → `bhuppi help net ports`, `systemctl` → launchctl, `journalctl` → `bhuppi help logs unified`). The reverse is the same. If the binary is actually on `PATH`, it runs. The platform note is for the missing case.

On Windows, Git for Windows `usr\bin` wins: the Unix binary stays the Unix binary. Without it, a short native map is used (`dir`, `type`, `findstr`, `tasklist` for a plain `ps`). `find.exe` and `sort.exe` in System32 are refused. They are not Unix `find` or `sort`.

`bhuppi doctor` lists what is present, labels GNU or BSD when it can tell, and does not install anything.

## explain

`bhuppi explain` is the A–Z index. Every Unix name in the atlas is a group, a leaf, a language-mode, a capability, a recipe, or an explicit skip.

```sh
bhuppi explain awk
```

```text
text awk
awk — filter, fields, begin-end, oneline, script
```

`bhuppi awk` is not a command. It exits 2 and tells you the job path.

## Recipes

Recipes print the pipeline. `--run` executes it. There is no shell string to inject.

```sh
bhuppi recipe disk-hogs .
bhuppi recipe --run checksum README
bhuppi recipe tar-dir logs
bhuppi recipe who-listens
bhuppi recipe newest-in .
bhuppi recipe cron-mine
bhuppi recipe conn-mine
```

Steps follow the host: GNU `du --max-depth` versus BSD `du -d`, `sha256sum` versus `shasum -a 256`, `ss` versus `lsof`.

## Not a cookbook for attack tools

These are named and refused, with no flags and no recipe: hashcat, john, hydra, metasploit, sqlmap, dig AXFR, surprise reboot/shutdown/halt/poweroff, hidden `rm -rf`.

`nmap`, `nc` / `netcat` / `ncat`, and macOS `security` (keychain) are docs-only: the name, the purpose, and the official manual. `bhuppi net nmap` does not scan.
