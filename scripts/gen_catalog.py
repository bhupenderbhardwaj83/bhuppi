#!/usr/bin/env python3
"""Generate internal/catalog/data.yaml.

Optional. Go embeds the YAML this script writes. Building bhuppi does not need Python.
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "internal" / "catalog" / "data.yaml"

RULE = "Wrap the real binary. Teach the shape. Never pretend GNU is BSD."


class Dumper(yaml.SafeDumper):
    def ignore_aliases(self, data):  # noqa: ANN001
        return True


def _str(dumper: Dumper, data: str):
    if "\n" in data:
        return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")
    return dumper.represent_scalar("tag:yaml.org,2002:str", data)


Dumper.add_representer(str, _str)


class Cat:
    def __init__(self) -> None:
        self.groups: list[dict] = []
        self.meta: list[dict] = []
        self.recipes: list[dict] = []
        self.explain: list[dict] = []
        self.nodes: dict[str, dict] = {}

    def group(self, name: str, title: str, what: str, when: str, rule: str, not_this: str, **kw) -> dict:
        node = dict(
            name=name,
            title=title,
            what=what,
            when=when,
            rule=rule,
            not_this=not_this,
            children=[],
            delivery="group",
        )
        node.update(kw)
        node.setdefault("desc", what)
        node.setdefault("more", f"bhuppi help {name} · bhuppi explain {name}")
        node.setdefault("shape", f"bhuppi {name} CMD [ARGS]")
        node.setdefault("example", f"bhuppi help {name}")
        self.groups.append(node)
        self.nodes[name] = node
        return node

    def add(self, path: str, title: str, what: str, **kw) -> dict:
        parts = path.split("/")
        if len(parts) < 2:
            raise SystemExit(f"path needs a group: {path}")
        parent = self.nodes.get("/".join(parts[:-1]))
        if parent is None:
            raise SystemExit(f"missing parent for {path}")
        name = parts[-1]
        job = path.replace("/", " ")
        bin_name = kw.get("bin") or ""
        if "unix" in kw:
            unix = kw["unix"]
            if isinstance(unix, str):
                unix = unix.split()
            elif unix is None:
                unix = []
        else:
            unix = []
        node = dict(name=name, title=title, what=what, children=[])
        node.update(kw)
        node["unix"] = unix
        shown = unix[0] if unix else (bin_name or name)
        node.setdefault("delivery", "passthrough")
        node.setdefault("user_args", "append")
        node.setdefault("when", f"You want {title} and the machine already has {shown}.")
        node.setdefault("rule", RULE)
        node.setdefault("desc", what + " --dry-run prints that argv and does not run it.")
        node.setdefault(
            "not_this",
            f"Not `bhuppi {shown}`. Not a second {shown}. GNU flags are not rewritten into BSD flags.",
        )
        node.setdefault("more", f"bhuppi help {job} · bhuppi explain {shown}")
        node.setdefault("shape", f"bhuppi {job} [ARGS]")
        node.setdefault("example", f"bhuppi {job}")
        if bin_name and "argv" not in node and node["delivery"] in ("passthrough", "alias", "peer"):
            if not node.get("special"):
                node["argv"] = [bin_name]
        parent.setdefault("children", []).append(node)
        self.nodes[path] = node
        return node

    def extra(self, name: str, target: str, line: str, delivery: str, what: str, **kw) -> None:
        row = dict(
            name=name,
            target=target,
            line=line,
            delivery=delivery,
            what=what,
            when=kw.get("when", "You typed a Unix name and want the job, or a reason bhuppi will not wrap it."),
            rule=kw.get("rule", RULE),
            not_this=kw.get("not_this", "Not a wrapper and not a flag recipe."),
            example=kw.get("example", ""),
            docs=kw.get("docs", ""),
        )
        self.explain.append(row)


def build() -> dict:
    c = Cat()
    identity = dict(
        name="bhuppi",
        wordmark="b h u p p i",
        tagline="parent atlas for Unix jobs",
        rule=RULE,
        what="One job tree that runs the Unix binary already on this machine.",
        when="You know the job — list, search, connect — and want the real argv, not a renamed coreutils.",
        not_this="Not a coreutils rewrite. Not `bhuppi ls` or `bhuppi awk`. Not a GNU-on-BSD flag translator. Not TACU: no model, no ask, no auto.",
        more="bhuppi help files · bhuppi explain awk · docs/tutorial.md",
    )

    # ---------------------------------------------------------------- groups
    c.group(
        "files",
        "everyday files",
        "Name, read, copy, move, and remove files by running the real ls, cat, cp, mv, and rm.",
        "The thing you mean is a file: show it, copy it, find it, or change its mode.",
        "File verbs exec the stock binary. rm never gains -rf unless you typed that flag.",
        "Not `bhuppi ls`. Not a trash can. files open is cat, the same job as files show, not the macOS open GUI. Empty directories go to rmdir; everything else is rm without new flags.",
        sparkle=True,
        pick="everyday files — list, show, copy, find",
        section="Sparkle",
    )
    c.group(
        "folders",
        "everyday directories",
        "Make, enter, and remove directories. Shared verbs use the same dispatcher as files; the help talks about folders.",
        "The thing you mean is a directory: go there, make it, or remove it when it is empty.",
        "folders go cds this shell through the hook. folders remove is rmdir and stays rmdir when the directory is not empty.",
        "Not a second path bucket glued to files. Not `open`. Not a recursive delete.",
        sparkle=True,
        pick="everyday directories — go, make, remove",
    )
    c.group(
        "text",
        "search and transform",
        "Search and rewrite text with rg or grep, awk, and sed. Language-modes teach a few shapes, then exec the real interpreter.",
        "You are filtering a log, cutting fields, or replacing a string.",
        "A safe default (awk fields, sed -i) is written on the page and printed by --dry-run. Other flags are yours.",
        "Not an awk textbook. Not `bhuppi awk`. Not a silent rewrite of GNU sed -i into the BSD form, or the reverse: inplace picks the flag for this OS and shows it.",
        sparkle=True,
        pick="search and transform — search, awk, sed",
    )
    c.group(
        "lang",
        "shell scripts",
        "Run a script with the real interpreter. Builtins are taught here and are not injected into your shell, except the files go / folders go hook.",
        "You have a bash, sh, or other script and want run, a syntax check, or a reminder that cd is a builtin.",
        "The child process execs bash, python, or perl. It cannot export, alias, or cd your parent shell.",
        "Not a new shell. Not eval of a string bhuppi invented. Not a login manager.",
    )
    c.group(
        "proc",
        "processes",
        "List processes, see who holds a command, path, or port, read one process's properties, and signal a pid you name.",
        "You want ps, lsof, or top, or you already know the pid to signal.",
        "kill stays kill. -9 is added only when you type it. A bare word to proc using is a command name, not a filename.",
        "Not a process hider. Not a credential dump. dtruss is a SIP skip, not a wrapper.",
        sparkle=True,
        pick="processes — list, using, top",
    )
    c.group(
        "net",
        "hosts you operate",
        "Fetch, ping, resolve, and ssh to hosts you administer. Filter pages only list rules.",
        "The machine is yours: a login, a health check, a listener, or a packet capture on an interface you admin.",
        "nmap and nc are docs-only and do not run. dig AXFR is refused. tcpdump is the safe shape: interface, -n, optional count.",
        "Not a scanner, pivot, or listener-as-backdoor. Not a firewall bypass. Not stealth capture.",
        sparkle=True,
        pick="hosts you operate — get, ping, ssh, ports",
    )
    c.group(
        "pack",
        "archives",
        "Create, list, extract, and compress archives with tar, gzip, zip, and the rest.",
        "You are packing a directory or opening an archive you already trust.",
        "tar compress is tar -czf. The dry-run argv is that command. Exclude-flag spelling stays on the help page, not a silent rewrite.",
        "Not an archive bomb helper. Not a second tar.",
    )
    c.group(
        "disk",
        "volumes",
        "See free space, directory weight, and (on macOS) diskutil list, info, and verify.",
        "You want df, du, or a read of the volume you already mounted.",
        "diskutil here is list, info, and verify. fdisk, when present, lists partitions and does not write them.",
        "Not a partition editor. Not an eraseDisk wrapper. Not a surprise unmount of the boot volume.",
    )
    c.group(
        "auth",
        "identity",
        "See who you are and, when you ask, re-run one command with sudo.",
        "You want who, id, or a single sudo command.",
        "sudo with no command prints help. It does not open a root shell for you.",
        "Not a password cracker. Not a sudoers editor. macOS security(1) keychain is docs-only.",
    )
    c.group(
        "svc",
        "services",
        "Ask systemd, launchd, or your user crontab about services you run.",
        "You want status, start, stop, enable, or your own crontab.",
        "systemctl is Linux-only and points at launchctl on macOS. launchctl load targets the gui user domain, units you own.",
        "Not a hunt for malicious units. Not a reboot. Not bootout of the system domain.",
    )
    c.group(
        "logs",
        "journals and sessions",
        "Read the journal, the kernel ring, or the unified log. The page is a reader, not a hunt.",
        "You want the last lines of a log you already administer.",
        "journalctl is Linux-only. The default is a short page, not a search for failed passwords.",
        "Not a brute-force hunt. Not a secret scrape.",
    )
    c.group(
        "host",
        "machine and packages",
        "Read the OS, the environment, and search or install a package with the manager this machine already uses.",
        "You want uname, env, or brew/apt/dnf for a package you named.",
        "The package manager is detected and the dry-run shows apt, dnf, yum, or brew. One is not rewritten into another.",
        "Not a new package manager. Not a silent sudo on brew.",
    )
    c.group(
        "data",
        "JSON, hash, and TLS",
        "Query JSON or YAML, hash a file, or run an openssl job you can read back: key, csr, cert text, verify, handshake.",
        "You have a file or a host you operate and want jq, a checksum, or a TLS handshake.",
        "openssl here is those five jobs. No intercepting proxy and no s_server listener.",
        "Not an interception kit. Not a hash cracker. md5 and md5sum stay peers: the page names which binary this OS runs.",
    )
    c.group(
        "dev",
        "builder",
        "Run git, the compiler, make, or docker that is already installed.",
        "You are in a repo or standing next to a binary you want ldd or otool to describe.",
        "Args are passed through. No extra mounts, capabilities, or flags are added to docker.",
        "Not a CI system. Not a debugger exploit recipe. dtruss is not offered.",
    )
    c.group(
        "docs",
        "look up",
        "Open the man page or ask which binary PATH would run.",
        "You want the manual, or the path of a tool, and then you will type the real command.",
        "man and which are exec'd as themselves. Docs-only tools stay on their own page and do not gain flags here.",
        "Not a copy of every man page. Not `bhuppi` plus a Unix name.",
    )

    files(c)
    folders(c)
    text(c)
    lang(c)
    proc(c)
    net(c)
    pack(c)
    disk(c)
    auth(c)
    svc(c)
    logs(c)
    host(c)
    data(c)
    dev(c)
    docs(c)
    meta(c)
    recipes(c)
    explain_extras(c)

    return {
        "identity": identity,
        "groups": c.groups,
        "meta": c.meta,
        "recipes": c.recipes,
        "explain": c.explain,
    }


def files(c: Cat) -> None:
    c.add(
        "files/list",
        "list names",
        "List names in a directory by running ls. --long is a long listing. --hidden is only names that start with a dot.",
        bin="ls",
        unix=["ls"],
        special="list",
        shape="bhuppi files list [PATH] [--long] [--hidden]",
        example="bhuppi files list .",
        examples=["bhuppi files list . --long", "bhuppi files list . --hidden", "bhuppi files list . --long --hidden"],
        when="You want the listing you already know from ls, a long listing, or only hidden names.",
        rule="Plain list is ls plus only the flags you typed. --long is ls -lah. --hidden is find for names that start with a dot, excluding . and .. themselves. --long --hidden is that find with -exec ls -lahd. Color flags are not rewritten.",
        desc="PATH defaults to . when you use --long or --hidden. A plain list does not inject -lah. --dry-run prints the argv that will run.",
        not_this="Not `bhuppi ls`. Not a GNU-on-BSD color translator. --hidden is not ls -a. Pass --color or -G yourself when you want color.",
        also_linux="ls --color=auto .",
        also_macos="ls -G .",
        runs="ls .",
        sparkle=True,
        section="Look",
        complete="both",
        min_args=0,
        user_args="none",
    )
    c.add(
        "files/show",
        "print a file",
        "Print a file by running cat.",
        bin="cat",
        unix=["cat"],
        shape="bhuppi files show FILE",
        example="bhuppi files show README",
        when="You want the bytes of one file on stdout.",
        rule="The argv is cat and the paths you named. No pager is inserted.",
        desc="FILE is required. Extra paths are extra cat arguments. Completion offers files and omits directories.",
        not_this="Not less, unless you asked for files less. Not a directory listing.",
        runs="cat README",
        sparkle=True,
        section="Look",
        complete="files",
        min_args=1,
    )
    c.add(
        "files/open",
        "print a file",
        "Print a file by running cat. Alias of files show.",
        bin="cat",
        unix=["open"],
        shape="bhuppi files open FILE",
        example="bhuppi files open README",
        when="You typed open and want the file on stdout, the same job as files show.",
        rule="The argv is cat and the paths you named. This is not the macOS open GUI.",
        desc="FILE is required. files open and files show both exec cat. --dry-run prints cat and the path.",
        not_this="Not the macOS open command. Not a GUI. Not less.",
        runs="cat README",
        explain_line="files open is an alias of files show. Both exec cat. Not the macOS open GUI.",
        section="Look",
        complete="files",
        min_args=1,
    )
    c.add(
        "files/map",
        "directory tree",
        "Draw a directory tree. find lists the entries. bhuppi draws the box-drawing tree.",
        bin="find",
        unix=[],
        special="map",
        shape="bhuppi files map [PATH] [--depth N]",
        example="bhuppi files map . --depth 2",
        when="You want a directory tree, not a flat listing.",
        rule="Default depth is 3. --depth N sets the levels. PATH defaults to . . Discovery is find PATH -maxdepth N. --dry-run prints that argv and a line that the drawing is bhuppi's tree view, not find's path list. Folders end with /. Each entry under the root is marked (Level N). The root line has no level tag.",
        desc="The same job as folders map. A plain files list is not a tree, and depth is not a hidden flag on list.",
        not_this="Not a flat find listing. Not files list. Do not hide depth inside files list.",
        runs="find . -maxdepth 3",
        also_linux="find . -maxdepth 3",
        also_macos="find . -maxdepth 3",
        section="Look",
        complete="folders",
        min_args=0,
        user_args="none",
    )
    c.add(
        "files/stat",
        "file properties",
        "Show properties of a file or folder: kind, size, mode, owner, group, links, inode, and the times this OS stat prints.",
        bin="stat",
        unix=["stat"],
        special="stat",
        shape="bhuppi files stat PATH",
        example="bhuppi files stat README",
        when="You want the Get Info view: kind, size, permissions, owner, and times.",
        rule="bhuppi chooses the native human-readable form. macOS runs stat -x PATH. Linux runs stat PATH. A flag you type is not rewritten from GNU to BSD. --os changes the help examples only. When PATH is a directory, du -sh PATH runs after stat.",
        desc="You will see kind, size, mode, owner, group, links, inode, and access, modify, change, and birth when this OS stat prints them. A directory also prints content size from du -sh.",
        not_this="Not ls -lah. files show stays cat. files open stays an alias of show.",
        runs="stat -x README",
        runs_linux="stat README",
        runs_darwin="stat -x README",
        also_linux="stat README",
        also_macos="stat -x README",
        explain_line="files stat shows properties with the native stat. folders stat is the same job for a directory.",
        section="Look",
        complete="both",
        min_args=1,
        user_args="none",
    )
    c.add(
        "files/where",
        "print the working directory",
        "Print the working directory by running pwd.",
        bin="pwd",
        unix=["pwd"],
        shape="bhuppi files where",
        example="bhuppi files where",
        when="You want the absolute directory of this shell.",
        rule="The argv is pwd. The shell's pwd builtin and /bin/pwd agree for ordinary directories.",
        desc="There is no PATH argument. --dry-run prints pwd.",
        not_this="Not folders where. where lives on files so the first-ten surface stays put.",
        runs="pwd",
        section="Look",
        complete="none",
        min_args=0,
        user_args="none",
        argv=["pwd"],
    )
    c.add(
        "files/go",
        "change directory",
        "cd into PATH in this shell. A child process cannot chdir its parent, so the installed hook runs cd.",
        bin="cd",
        unix=["cd"],
        special="cd",
        delivery="alias",
        shape="bhuppi files go PATH",
        example="bhuppi files go src",
        when="You want the files verb for cd, including the locked first-ten surface.",
        rule="The binary prints the shape and refuses to pretend it moved you. The bash/zsh hook, or bhuppi.cmd, runs cd.",
        desc="--dry-run prints cd PATH and does not change directory. After install, a new shell sources the hook so files go and folders go cd for real. Builtins other than this hook are not injected.",
        not_this="Not a new shell. Not pushd. export, alias, history, type, and test stay builtins.",
        runs="cd src",
        section="Change",
        complete="folders",
        min_args=1,
        user_args="none",
    )
    c.add(
        "files/make",
        "make a file or directory",
        "mkdir a directory, or touch a file when you pass --empty.",
        bin="mkdir",
        unix=["mkdir", "touch"],
        special="make",
        shape="bhuppi files make DIR|FILE [--empty]",
        example="bhuppi files make tmp",
        examples=["bhuppi files make a --empty"],
        when="You want a new directory, or an empty file.",
        rule="--empty selects touch. Anything else is mkdir. -p is passed only when you type it.",
        desc="bhuppi files make tmp runs mkdir tmp. bhuppi files make a --empty runs touch a. --dry-run prints that argv.",
        not_this="Not a template engine. Not install(1) unless you asked for files install.",
        runs="mkdir tmp",
        runs_darwin="touch a",
        also_linux="touch a",
        also_macos="touch a",
        section="Change",
        complete="both",
        min_args=1,
    )
    c.add(
        "files/copy",
        "copy a file",
        "Copy SRC to DST by running cp.",
        bin="cp",
        unix=["cp"],
        shape="bhuppi files copy SRC DST",
        example="bhuppi files copy a b",
        when="You want a second copy and the original left in place.",
        rule="The argv is cp plus your arguments. -r is added only when you type it.",
        desc="Two paths is the ordinary shape. Flags you type, such as -n or -r, are passed through and shown by --dry-run.",
        not_this="Not move. Not a backup policy.",
        runs="cp a b",
        section="Change",
        complete="both",
        min_args=2,
    )
    c.add(
        "files/move",
        "move or rename",
        "Move or rename SRC to DST by running mv.",
        bin="mv",
        unix=["mv"],
        shape="bhuppi files move SRC DST",
        example="bhuppi files move a b",
        when="You want the name changed or the file in another directory.",
        rule="The argv is mv plus your arguments. Nothing is copied first.",
        desc="Cross-device moves are mv's own business. --dry-run prints mv SRC DST.",
        not_this="Not copy. Not a trash can.",
        runs="mv a b",
        section="Change",
        complete="both",
        min_args=2,
    )
    c.add(
        "files/remove",
        "remove a file or empty directory",
        "Remove PATH. An empty directory is rmdir. Anything else is rm, with only the flags you typed.",
        bin="rm",
        unix=["rm"],
        special="remove",
        shape="bhuppi files remove PATH",
        example="bhuppi files remove a",
        examples=["bhuppi files remove empty-dir"],
        when="You want a file gone, or an empty directory removed.",
        rule="Never add -r, -f, or -rf unless you typed it. An empty directory selects rmdir so rm is not given a directory by surprise.",
        desc="One empty directory runs rmdir. A file runs rm with your flags only. A non-empty directory is left to rm, which refuses without -r you typed yourself.",
        not_this="Not shred. Not a hidden rm -rf. Not folders remove, which is always rmdir.",
        runs="rm a",
        also_linux="rmdir empty-dir",
        also_macos="rmdir empty-dir",
        section="Change",
        complete="both",
        min_args=1,
    )
    c.add(
        "files/find",
        "find by name",
        "Find by a glob, or by a simplified name shape. fd --glob when fd is on PATH, otherwise find -name. One syntax either way.",
        bin="find",
        prefer=["fd", "find"],
        unix=["find", "fd"],
        special="find",
        delivery="language-mode",
        shape="bhuppi files find NAME [PATH]",
        example="bhuppi files find '*.go' .",
        examples=[
            "bhuppi files find . --contains go",
            "bhuppi files find . --startswith main",
            "bhuppi files find . --endswith .go",
            "bhuppi files find . --anyof a-z",
            "bhuppi files find . --oneof a,b,c",
            "bhuppi files find . --regex '.*\\.go'",
            "bhuppi files find . --contains readme --sensitivecase",
            "bhuppi files find . --contains readme --insensitivecase",
            "bhuppi files find readme . --word",
            "bhuppi files find . --contains readme --invert",
        ],
        when="You know the glob and the tree to walk, or you want a simple contains, starts-with, ends-with, class, or list of names.",
        rule="With no shape flag the pattern is a shell glob: fd --glob when fd is on PATH, otherwise find -name. --contains TEXT is find -name '*TEXT*'. --startswith TEXT is 'TEXT*'. --endswith TEXT is '*TEXT'. --anyof a-z is a character class, '*[a-z]*', and the class moves to the anchor when you also pass contains, startswith, or endswith. --oneof a,b,c is find ( -name a -o -name b -o -name c ), with the same anchor on each value. --regex PATTERN is your regex, not a translation. On this Mac that is find PATH -regex PATTERN, which is BSD find and matches the whole path. --sensitivecase is the default and keeps -name. --insensitivecase uses -iname. --regex with --insensitivecase is find PATH -iregex PATTERN; the pattern is not rewritten. --word on find means the file name is exactly TEXT (-name TEXT). With --contains, find has no word-boundary flag, so the pattern stays '*TEXT*'. --invert is ! -name or ! -iname. --line is for text search. --dry-run prints the argv that will actually run.",
        desc="NAME then optional PATH when you are using a glob. With a shape flag, the positional is the path and it defaults to `.`. Quote globs so your shell does not expand them. Children name, type, size, time, and exec call find.",
        not_this="Not Windows find.exe. Not a content search; that is text search. These flags are not a second regex language; --regex is still the raw pattern.",
        runs="find . -name '*.go'",
        also_linux="find . -regextype posix-extended -regex '.*\\.go'",
        also_macos="find . -regex '.*\\.go'",
        sparkle=True,
        section="Look",
        complete="both",
        min_args=1,
    )
    for name, title, what, shape, example, special, runs, skip in [
        (
            "name",
            "find by glob",
            "Same glob as files find, spelled as the name child so the tree is explicit.",
            "bhuppi files find name NAME [PATH]",
            "bhuppi files find name '*.go' .",
            "find",
            "find . -name '*.go'",
            0,
        ),
        (
            "type",
            "find by type",
            "Restrict the walk to a find type letter: f, d, l, and the rest find accepts.",
            "bhuppi files find type TYPE [PATH]",
            "bhuppi files find type f .",
            "find-type",
            "find . -type f",
            0,
        ),
        (
            "size",
            "find by size",
            "Pass a find -size expression such as +1M or -10k.",
            "bhuppi files find size SPEC [PATH]",
            "bhuppi files find size +1M .",
            "find-size",
            "find . -size +1M",
            0,
        ),
        (
            "time",
            "find by mtime",
            "Pass a find -mtime number such as -1 for the last day.",
            "bhuppi files find time N [PATH]",
            "bhuppi files find time -1 .",
            "find-time",
            "find . -mtime -1",
            0,
        ),
        (
            "exec",
            "find and exec",
            "Run a command on each match. {} and ; are added when you leave them out.",
            "bhuppi files find exec NAME PATH CMD...",
            "bhuppi files find exec '*.log' . ls -l",
            "find-exec",
            "find . -name '*.log' -exec ls -l {} ;",
            2,
        ),
    ]:
        c.add(
            f"files/find/{name}",
            title,
            what,
            bin="find",
            special=special,
            delivery="passthrough",
            shape=shape,
            example=example,
            runs=runs,
            also_linux="find -printf is GNU-only; this child uses -name, -type, -size, -mtime, or -exec.",
            also_macos="BSD find has no -printf. The argv on this page is the portable form.",
            section="Find",
            complete="both" if name != "exec" else "none",
            complete_skip=skip,
            min_args=1 if name != "exec" else 3,
            user_args="none",
        )

    simple_files = [
        ("file", "file type", "file", "file", "Classify a file by running file(1).", "bhuppi files file FILE", "bhuppi files file README", "files", 1, {}),
        ("chmod", "change mode", "chmod", "chmod", "Change mode bits by running chmod.", "bhuppi files chmod MODE PATH", "bhuppi files chmod 644 README", "both", 2, {}),
        ("chown", "change owner", "chown", "chown", "Change owner by running chown.", "bhuppi files chown OWNER PATH", "bhuppi files chown user README", "both", 2, {}),
        ("chgrp", "change group", "chgrp", "chgrp", "Change group by running chgrp.", "bhuppi files chgrp GROUP PATH", "bhuppi files chgrp staff README", "both", 2, {}),
        ("link", "make a link", "ln", "ln", "Make a hard link, or pass -s yourself for a symlink.", "bhuppi files link SRC DST", "bhuppi files link -s src dst", "both", 2,
         {"desc": "The argv is ln plus your arguments. -s is not implied. --dry-run prints ln and those arguments."}),
        ("readlink", "read a symlink", "readlink", "readlink", "Print the target of a symlink by running readlink.", "bhuppi files readlink PATH", "bhuppi files readlink link", "files", 1, {}),
        ("realpath", "canonical path", "realpath", "realpath", "Print the canonical path by running realpath.", "bhuppi files realpath PATH", "bhuppi files realpath .", "both", 1, {}),
        ("basename", "strip directory", "basename", "basename", "Print the last path component by running basename.", "bhuppi files basename PATH", "bhuppi files basename src/main.go", "none", 1, {}),
        ("dirname", "strip last component", "dirname", "dirname", "Print the directory portion by running dirname.", "bhuppi files dirname PATH", "bhuppi files dirname src/main.go", "none", 1, {}),
        ("mktemp", "temporary path", "mktemp", "mktemp", "Create a temporary file by running mktemp.", "bhuppi files mktemp", "bhuppi files mktemp", "none", 0,
         {"desc": "With no arguments the argv is mktemp, which creates a file in the temp directory. Extra mktemp flags are passed through."}),
        ("less", "page a file", "less", "less", "Page a file by running less.", "bhuppi files less FILE", "bhuppi files less README", "files", 1, {}),
        ("more", "page a file", "more", "more", "Page a file by running more.", "bhuppi files more FILE", "bhuppi files more README", "files", 1, {}),
        ("rsync", "sync trees", "rsync", "rsync", "Copy trees by running rsync with the arguments you typed.", "bhuppi files rsync SRC DST", "bhuppi files rsync -a src/ dst/", "both", 2, {}),
        ("install", "install a file", "install", "install", "Copy a file into place by running install(1), the Unix tool, not the bhuppi installer.", "bhuppi files install SRC DST", "bhuppi files install a /tmp/a", "both", 2,
         {"not_this": "Not ./install, which installs bhuppi itself. This page runs the Unix install binary."}),
    ]
    for name, title, bin_name, unix, what, shape, example, comp, min_args, extra in simple_files:
        kw = dict(bin=bin_name, unix=[unix], shape=shape, example=example, section="Meta", complete=comp, min_args=min_args)
        kw.update(extra)
        c.add(f"files/{name}", title, what, **kw)


def folders(c: Cat) -> None:
    c.add(
        "folders/go",
        "enter a directory",
        "cd into DIR in this shell. This is the folder-primary spelling of the same hook as files go.",
        bin="cd",
        unix=[],
        special="cd",
        delivery="alias",
        shape="bhuppi folders go DIR",
        example="bhuppi folders go src",
        when="The thing you are entering is a directory.",
        rule="Same hook as files go. The binary alone cannot chdir the parent.",
        desc="--dry-run prints cd DIR. Completion offers directories and omits files.",
        not_this="Not files where. Not pushd.",
        runs="cd src",
        sparkle=True,
        section="Change",
        complete="folders",
        min_args=1,
    )
    c.add(
        "folders/make",
        "make a directory",
        "Create a directory by running mkdir.",
        bin="mkdir",
        unix=[],
        special="mkdir",
        shape="bhuppi folders make DIR",
        example="bhuppi folders make tmp",
        when="You want a directory, not an empty file.",
        rule="--empty is refused here. Empty files are files make --empty.",
        desc="The argv is mkdir plus your arguments. -p is passed only when you type it.",
        not_this="Not touch. Not files make --empty.",
        runs="mkdir tmp",
        sparkle=True,
        section="Change",
        complete="folders",
        min_args=1,
    )
    c.add(
        "folders/remove",
        "remove an empty directory",
        "Remove an empty directory by running rmdir.",
        bin="rmdir",
        unix=["rmdir"],
        special="rmdir",
        shape="bhuppi folders remove DIR",
        example="bhuppi folders remove tmp",
        when="The directory is empty and you want it gone.",
        rule="Always rmdir. A non-empty directory makes rmdir fail. bhuppi does not switch to rm -rf.",
        desc="--dry-run prints rmdir DIR. To remove files, use files remove and type -r yourself if you mean it.",
        not_this="Not rm -rf. Not files remove.",
        runs="rmdir tmp",
        section="Change",
        complete="folders",
        min_args=1,
    )
    c.add(
        "folders/list",
        "list a directory",
        "List a directory by running ls. Same dispatcher as files list; this page talks about folders.",
        bin="ls",
        unix=[],
        special="list",
        shape="bhuppi folders list [DIR] [--long] [--hidden]",
        example="bhuppi folders list .",
        examples=["bhuppi folders list . --long", "bhuppi folders list . --hidden"],
        when="You are looking at directories rather than file contents.",
        rule="Same argv as files list. --long is ls -lah. --hidden is find for dot names, excluding . and .. themselves. Color flags are still not rewritten.",
        desc="DIR defaults to . when you use --long or --hidden. A plain list does not inject -lah.",
        not_this="Not a different ls. Not files show. --hidden is not ls -a.",
        also_linux="ls --color=auto .",
        also_macos="ls -G .",
        runs="ls .",
        section="Look",
        complete="folders",
        min_args=0,
        user_args="none",
    )
    c.add(
        "folders/map",
        "directory tree",
        "Draw a directory tree. find lists the entries. bhuppi draws the box-drawing tree.",
        bin="find",
        unix=[],
        special="map",
        shape="bhuppi folders map [PATH] [--depth N]",
        example="bhuppi folders map . --depth 2",
        when="You want a directory tree, not a flat listing.",
        rule="Default depth is 3. --depth N sets the levels. PATH defaults to . . Discovery is find PATH -maxdepth N. --dry-run prints that argv and a line that the drawing is bhuppi's tree view, not find's path list. Folders end with /. Each entry under the root is marked (Level N). The root line has no level tag.",
        desc="The same job as files map. A plain files list is not a tree, and depth is not a hidden flag on list.",
        not_this="Not a flat find listing. Not files list. Do not hide depth inside files list.",
        runs="find . -maxdepth 3",
        also_linux="find . -maxdepth 3",
        also_macos="find . -maxdepth 3",
        section="Look",
        complete="folders",
        min_args=0,
        user_args="none",
    )
    c.add(
        "folders/stat",
        "folder properties",
        "Show properties of a folder: kind, size, mode, owner, group, links, inode, and the times this OS stat prints. Content size follows from du.",
        bin="stat",
        unix=[],
        special="stat",
        shape="bhuppi folders stat PATH",
        example="bhuppi folders stat .",
        when="You want the Get Info view for a directory.",
        rule="Same dispatcher as files stat. macOS runs stat -x PATH, then du -sh PATH because a directory also reports content size. Linux runs stat PATH, then du -sh PATH. --os changes help examples only.",
        desc="You will see kind, size, mode, owner, group, links, inode, and access, modify, change, and birth when this OS stat prints them. du -sh prints the content size on its own.",
        not_this="Not ls -lah. Not files show.",
        runs="stat -x .",
        runs_linux="stat .",
        runs_darwin="stat -x .",
        also_linux="stat .",
        also_macos="stat -x .",
        section="Look",
        complete="folders",
        min_args=1,
        user_args="none",
    )
    c.add(
        "folders/copy",
        "copy a directory",
        "Copy with cp. Same dispatcher as files copy. Pass -r yourself when you mean a tree.",
        bin="cp",
        unix=[],
        shape="bhuppi folders copy SRC DST",
        example="bhuppi folders copy -r src dst",
        when="You want a directory copied and you are willing to type -r.",
        rule="-r is not injected.",
        desc="The argv is cp plus your arguments. --dry-run shows them unchanged.",
        not_this="Not rsync, unless you asked for files rsync.",
        runs="cp -r src dst",
        section="Change",
        complete="both",
        min_args=2,
    )
    c.add(
        "folders/move",
        "move a directory",
        "Move a directory by running mv. Same dispatcher as files move.",
        bin="mv",
        unix=[],
        shape="bhuppi folders move SRC DST",
        example="bhuppi folders move src dst",
        when="You want a directory renamed or relocated.",
        rule="The argv is mv plus your arguments.",
        desc="Same binary as files move. The help on this page is about directories.",
        not_this="Not copy.",
        runs="mv src dst",
        section="Change",
        complete="both",
        min_args=2,
    )
    c.add(
        "folders/find",
        "find directories",
        "Find with the same dispatcher as files find. Use the type child when you want only directories.",
        bin="find",
        prefer=["fd", "find"],
        unix=[],
        special="find",
        delivery="language-mode",
        shape="bhuppi folders find NAME [PATH]",
        example="bhuppi folders find 'src*' .",
        when="You are hunting directories by glob, or you want find's type, size, time, or exec children.",
        rule="One glob syntax. fd --glob if fd exists, otherwise find -name. The same shape flags as files find apply here: --contains, --startswith, --endswith, --anyof, --oneof, --regex, --sensitivecase, --insensitivecase, --word, and --invert. --line is for text search. Children always call find.",
        desc="This is not a separate finder. The help says folder because that is the job you named.",
        not_this="Not Windows find.exe. Not a content search. These flags are not a second regex language; --regex is still the raw pattern.",
        runs="find . -name 'src*'",
        section="Look",
        complete="folders",
        min_args=1,
    )
    for name, title, what, shape, example, special, runs, min_args in [
        ("name", "find by glob", "Glob search, same as files find name.", "bhuppi folders find name NAME [PATH]", "bhuppi folders find name 'src*' .", "find", "find . -name 'src*'", 1),
        ("type", "find by type", "find -type. Use d when you want only directories.", "bhuppi folders find type TYPE [PATH]", "bhuppi folders find type d .", "find-type", "find . -type d", 1),
        ("size", "find by size", "find -size with the spec you type.", "bhuppi folders find size SPEC [PATH]", "bhuppi folders find size +1M .", "find-size", "find . -size +1M", 1),
        ("time", "find by mtime", "find -mtime with the number you type.", "bhuppi folders find time N [PATH]", "bhuppi folders find time -1 .", "find-time", "find . -mtime -1", 1),
        ("exec", "find and exec", "find -exec for a command you name.", "bhuppi folders find exec NAME PATH CMD...", "bhuppi folders find exec 'src*' . ls -ld", "find-exec", "find . -name 'src*' -exec ls -ld {} ;", 3),
    ]:
        c.add(
            f"folders/find/{name}",
            title,
            what,
            bin="find",
            special=special,
            shape=shape,
            example=example,
            runs=runs,
            min_args=min_args,
            complete="folders",
            user_args="none",
            section="Find",
        )
    c.add(
        "folders/tree",
        "draw a tree",
        "Draw a directory tree by running tree, when tree is installed.",
        bin="tree",
        unix=["tree"],
        shape="bhuppi folders tree [DIR]",
        example="bhuppi folders tree .",
        when="You want the tree(1) picture, not a flat ls.",
        hint="brew install tree    or    apt install tree",
        section="Look",
        complete="folders",
        min_args=0,
        optional=True,
    )


def text(c: Cat) -> None:
    c.add(
        "text/search",
        "search file contents",
        "Search text. rg when it is on PATH, otherwise grep -n. The default is a literal search.",
        bin="grep",
        prefer=["rg", "grep"],
        unix=["grep", "egrep", "fgrep", "rg"],
        special="search",
        shape="bhuppi text search TEXT [PATH]",
        example="bhuppi text search TODO src",
        examples=[
            "bhuppi text search --contains TODO src",
            "bhuppi text search --startswith TODO src",
            "bhuppi text search --endswith TODO src",
            "bhuppi text search --anyof a-z src",
            "bhuppi text search --oneof TODO,FIXME src",
            "bhuppi text search --regex 'TO.O' src",
            "bhuppi text search readme . --sensitivecase",
            "bhuppi text search readme . --insensitivecase",
            "bhuppi text search readme . --word",
            "bhuppi text search readme . --invert",
            "bhuppi text search readme . --line",
        ],
        when="You want lines that contain TEXT, or a simple anchored, class, or list pattern.",
        rule="The default is a literal search: rg -F, or grep -n -F. --contains is that same literal. --startswith and --endswith anchor a literal. --anyof and --oneof build a simple pattern. --regex passes your pattern to rg, or to grep -E, as a real regex. --sensitivecase is the default: rg -s, so smart-case stays off, or grep with no -i. --insensitivecase is rg -i or grep -i, including with --regex, and the pattern is not rewritten. --word is a word boundary, rg -w or grep -w. --invert is rg -v or grep -v. --line matches the whole line, rg -x or grep -x. A missing PATH becomes `.` for grep so it searches here instead of waiting on stdin. --dry-run shows which binary will run.",
        desc="Quote the text when it has spaces. --glob and other flags belong to the underlying tool: pass them only if you know which binary will run, and read the dry-run first.",
        not_this="Not a second grep. GNU grep -P is not implied and is not translated. These flags are not a second regex language; --regex is still the raw pattern.",
        runs="rg -F TODO src",
        also_linux="grep -n -F TODO src",
        also_macos="grep -n -F TODO src",
        sparkle=True,
        section="Search",
        complete="both",
        complete_skip=1,
        min_args=1,
    )
    c.add(
        "text/awk",
        "awk",
        "Filter and cut text with awk. Each child execs awk. This is a map, not a textbook.",
        bin="awk",
        unix=["awk"],
        delivery="language-mode",
        shape="bhuppi text awk filter|fields|begin-end|oneline|script …",
        example="bhuppi text awk fields 1,3 a.log",
        when="You want fields, a pattern, or a one-file awk program.",
        rule="The child builds the awk program it documents. --dry-run prints that program. GNU-only awk extensions are not added on BSD awk.",
        desc="Five children: filter, fields, begin-end, oneline, script. Bare `bhuppi text awk` prints this page.",
        not_this="Not `bhuppi awk`. Not a language tutorial. Not gawk-only features disguised as portable awk.",
        sparkle=True,
        section="Transform",
        complete="none",
        min_args=0,
        user_args="none",
    )
    c.add(
        "text/awk/filter",
        "filter lines",
        "Keep lines matching a pattern by running awk '/PAT/'.",
        bin="awk",
        special="awk-filter",
        shape="bhuppi text awk filter PAT [FILE]",
        example="bhuppi text awk filter TODO app.log",
        when="You want a simple pattern filter and you want to see the awk program.",
        rule="PAT is placed in an awk /pattern/. Slashes in PAT are escaped. The argv is awk and that program.",
        desc="FILE is optional; without it awk reads stdin. --dry-run prints the program.",
        not_this="Not text search. Not a regex cookbook.",
        runs="awk /TODO/ app.log",
        complete="files",
        complete_skip=1,
        min_args=1,
        user_args="none",
    )
    c.add(
        "text/awk/fields",
        "print fields",
        "Print chosen columns. fields 1,3 becomes awk '{print $1,$3}'.",
        bin="awk",
        special="awk-fields",
        shape="bhuppi text awk fields FIELDS FILE",
        example="bhuppi text awk fields 1,3 a.log",
        when="You want $1, $3, or any comma-separated field list.",
        rule="The default taught here is 1,3. Whatever list you type is the print list in the dry-run argv. No other awk is added.",
        desc="FIELDS is a comma-separated list of field numbers. FILE is the input. The program is exactly {print $N,...} with those numbers.",
        not_this="Not cut, unless you want text cut. Not a general awk tutorial.",
        runs="awk '{print $1,$3}' a.log",
        complete="files",
        complete_skip=1,
        min_args=2,
        user_args="none",
    )
    c.add(
        "text/awk/begin-end",
        "BEGIN and END",
        "Run one awk program you write, typically with BEGIN and END.",
        bin="awk",
        special="awk-program",
        shape="bhuppi text awk begin-end PROGRAM [FILE]",
        example="bhuppi text awk begin-end 'BEGIN{c=0} {c++} END{print c}' a.log",
        when="You already have the one-liner, including BEGIN or END.",
        rule="PROGRAM is passed to awk unchanged. bhuppi does not wrap it in another script.",
        desc="This child exists so the tree has a place for BEGIN/END. The argv is awk, your program, and optional files.",
        not_this="Not a tutorial on BEGIN and END.",
        runs="awk 'BEGIN{c=0} {c++} END{print c}' a.log",
        complete="files",
        complete_skip=1,
        min_args=1,
        user_args="none",
    )
    c.add(
        "text/awk/oneline",
        "one-line program",
        "Run one awk program you write.",
        bin="awk",
        special="awk-program",
        shape="bhuppi text awk oneline PROGRAM [FILE]",
        example="bhuppi text awk oneline 'NR==1' a.log",
        when="The whole program fits in one argument.",
        rule="PROGRAM is argv to awk, unchanged.",
        desc="Use fields when you only want columns. Use oneline when the program is yours.",
        not_this="Not a REPL.",
        runs="awk 'NR==1' a.log",
        complete="files",
        complete_skip=1,
        min_args=1,
        user_args="none",
    )
    c.add(
        "text/awk/script",
        "run an awk file",
        "Run an awk script file with awk -f.",
        bin="awk",
        special="awk-script",
        shape="bhuppi text awk script SCRIPT [FILE]",
        example="bhuppi text awk script prog.awk a.log",
        when="The program lives in a file.",
        rule="The argv is awk -f SCRIPT and the files you name.",
        desc="SCRIPT is required. Further arguments are files for awk.",
        not_this="Not an editor for the script.",
        runs="awk -f prog.awk a.log",
        complete="files",
        min_args=1,
        user_args="none",
    )
    c.add(
        "text/sed",
        "sed",
        "Replace, delete, or edit in place with sed. Three children, then the real sed.",
        bin="sed",
        unix=["sed"],
        delivery="language-mode",
        shape="bhuppi text sed replace|delete|inplace …",
        example="bhuppi text sed inplace foo bar f",
        when="You want a substitution or a delete, and you want the sed script visible.",
        rule="inplace uses -i on GNU sed and -i '' on BSD sed. --dry-run prints the form this OS will run. It does not rewrite a script you already wrote.",
        desc="replace and delete print to stdout. inplace edits the file. Bare `bhuppi text sed` prints this page.",
        not_this="Not `bhuppi sed`. Not a silent swap of GNU and BSD in-place flags inside a script you passed through.",
        also_linux="sed -i 's/foo/bar/g' f",
        also_macos="sed -i '' 's/foo/bar/g' f",
        section="Transform",
        complete="none",
    )
    c.add(
        "text/sed/replace",
        "replace text",
        "Print a substitution. The file is not modified.",
        bin="sed",
        special="sed-replace",
        shape="bhuppi text sed replace FROM TO FILE",
        example="bhuppi text sed replace foo bar f",
        when="You want stdout, not an edited file.",
        rule="The argv is sed and s/FROM/TO/g. The same script works on GNU and BSD sed because there is no -i.",
        desc="FROM and TO are literal text, with the delimiter escaped. FILE is required.",
        not_this="Not inplace. Not a stream editor tutorial.",
        runs="sed s/foo/bar/g f",
        complete="files",
        complete_skip=2,
        min_args=3,
        user_args="none",
    )
    c.add(
        "text/sed/delete",
        "delete matching lines",
        "Drop lines that match PAT. The file is not modified.",
        bin="sed",
        special="sed-delete",
        shape="bhuppi text sed delete PAT FILE",
        example="bhuppi text sed delete '^#' f",
        when="You want matching lines removed from the stream.",
        rule="The argv is sed and /PAT/d. PAT is escaped, not interpreted by a shell.",
        desc="Output is stdout. Use inplace only when you mean to edit the file, and that child is a substitution, not this delete.",
        not_this="Not rm. Not inplace.",
        runs="sed '/^#/d' f",
        complete="files",
        complete_skip=1,
        min_args=2,
        user_args="none",
    )
    c.add(
        "text/sed/inplace",
        "edit a file in place",
        "Substitute FROM with TO in FILE. GNU sed gets -i. BSD sed gets -i ''.",
        bin="sed",
        special="sed-inplace",
        shape="bhuppi text sed inplace FROM TO FILE",
        example="bhuppi text sed inplace foo bar f",
        when="You want the file changed and you want the OS-honest -i form.",
        rule="The flag is chosen for this operating system and printed by --dry-run. It is not a hidden rewrite of a sed script you authored yourself.",
        desc="On Linux the argv is sed -i s/FROM/TO/g FILE. On macOS it is sed -i '' s/FROM/TO/g FILE. Backup suffixes are not invented.",
        not_this="Not a cross-editor. Not GNU -i on a Mac, and not BSD -i '' on Linux.",
        runs_linux="sed -i s/foo/bar/g f",
        runs_darwin="sed -i '' s/foo/bar/g f",
        also_linux="sed -i 's/foo/bar/g' f",
        also_macos="sed -i '' 's/foo/bar/g' f",
        complete="files",
        complete_skip=2,
        min_args=3,
        user_args="none",
    )
    rows = [
        ("first", "first lines", "head", "head", "alias", "Print the first lines by running head.", "bhuppi text first [FILE]", "bhuppi text first README", "files", 0),
        ("last", "last lines", "tail", "tail", "alias", "Print the last lines by running tail.", "bhuppi text last [FILE]", "bhuppi text last README", "files", 0),
        ("cut", "cut fields", "cut", "cut", "passthrough", "Cut columns by running cut with the flags you type.", "bhuppi text cut ARGS", "bhuppi text cut -f1,3 -d, a.csv", "files", 1),
        ("sort", "sort lines", "sort", "sort", "passthrough", "Sort lines by running Unix sort.", "bhuppi text sort [FILE]", "bhuppi text sort README", "files", 0),
        ("uniq", "unique lines", "uniq", "uniq", "passthrough", "Drop adjacent duplicates by running uniq.", "bhuppi text uniq [FILE]", "bhuppi text uniq names.txt", "files", 0),
        ("tr", "translate characters", "tr", "tr", "passthrough", "Translate characters by running tr.", "bhuppi text tr SET1 SET2", "bhuppi text tr a-z A-Z", "none", 2),
        ("wc", "count lines", "wc", "wc", "passthrough", "Count lines, words, or bytes by running wc.", "bhuppi text wc [FILE]", "bhuppi text wc -l README", "files", 0),
        ("diff", "compare files", "diff", "diff", "passthrough", "Compare files by running diff.", "bhuppi text diff A B", "bhuppi text diff a b", "files", 2),
        ("cmp", "compare bytes", "cmp", "cmp", "passthrough", "Compare bytes by running cmp.", "bhuppi text cmp A B", "bhuppi text cmp a b", "files", 2),
        ("tee", "copy stdin to a file", "tee", "tee", "passthrough", "Copy stdin to a file and stdout by running tee.", "bhuppi text tee FILE", "bhuppi text tee out.txt", "files", 1),
        ("paste", "merge lines", "paste", "paste", "passthrough", "Merge lines of files by running paste.", "bhuppi text paste FILES", "bhuppi text paste a b", "files", 1),
        ("comm", "compare sorted lines", "comm", "comm", "passthrough", "Compare two sorted files by running comm.", "bhuppi text comm A B", "bhuppi text comm a b", "files", 2),
        ("column", "columnate", "column", "column", "passthrough", "Columnate lines by running column.", "bhuppi text column [FILE]", "bhuppi text column -t rows.txt", "files", 0),
        ("expand", "tabs to spaces", "expand", "expand", "passthrough", "Turn tabs into spaces by running expand.", "bhuppi text expand [FILE]", "bhuppi text expand Makefile", "files", 0),
        ("unexpand", "spaces to tabs", "unexpand", "unexpand", "passthrough", "Turn spaces into tabs by running unexpand.", "bhuppi text unexpand [FILE]", "bhuppi text unexpand README", "files", 0),
        ("fmt", "wrap paragraphs", "fmt", "fmt", "passthrough", "Wrap paragraphs by running fmt.", "bhuppi text fmt [FILE]", "bhuppi text fmt README", "files", 0),
        ("fold", "fold long lines", "fold", "fold", "passthrough", "Fold long lines by running fold.", "bhuppi text fold [FILE]", "bhuppi text fold -s -w 80 README", "files", 0),
        ("nl", "number lines", "nl", "nl", "passthrough", "Number lines by running nl.", "bhuppi text nl [FILE]", "bhuppi text nl README", "files", 0),
        ("rev", "reverse lines", "rev", "rev", "passthrough", "Reverse each line by running rev.", "bhuppi text rev [FILE]", "bhuppi text rev README", "files", 0),
        ("tac", "reverse line order", "tac", "tac", "passthrough", "Print lines last-to-first by running tac (usually Linux).", "bhuppi text tac FILE", "bhuppi text tac README", "files", 1),
        ("strings", "printable strings", "strings", "strings", "passthrough", "Print printable strings by running strings.", "bhuppi text strings FILE", "bhuppi text strings /bin/ls", "files", 1),
        ("xargs", "build argument lists", "xargs", "xargs", "passthrough", "Build argument lists by running xargs with the command you name.", "bhuppi text xargs CMD", "bhuppi text xargs ls", "none", 0),
        ("split", "split a file", "split", "split", "passthrough", "Split a file into pieces by running split.", "bhuppi text split FILE", "bhuppi text split -l 100 big.txt", "files", 1),
    ]
    for name, title, bin_name, unix, delivery, what, shape, example, comp, min_args in rows:
        extra = {}
        if name == "sort":
            extra["not_this"] = "Not Windows sort.exe. On Windows, Unix sort runs only from Git usr\\bin; otherwise bhuppi refuses."
            extra["hint"] = "Install Git for Windows so usr\\bin\\sort.exe is the Unix sort."
        if name == "tac":
            extra["platform"] = "linux"
            extra["fallback"] = "text last"
            extra["hint"] = "tac is typically GNU/Linux. On macOS use tail -r yourself; bhuppi does not rewrite tac into tail -r."
        c.add(
            f"text/{name}",
            title,
            what,
            bin=bin_name,
            unix=[unix],
            delivery=delivery,
            shape=shape,
            example=example,
            section="Lines",
            complete=comp,
            min_args=min_args,
            **extra,
        )


def lang(c: Cat) -> None:
    c.add(
        "lang/bash",
        "bash",
        "Run bash, check a script, or read which builtins bhuppi will not inject.",
        bin="bash",
        unix=["bash"],
        delivery="language-mode",
        shape="bhuppi lang bash run|check|flags|builtin …",
        example="bhuppi lang bash run script.sh",
        when="You have a bash script or you want bash -n.",
        rule="Each child execs bash except builtin, which is help only.",
        desc="run is bash SCRIPT. check is bash -n. flags passes the arguments you type. builtin explains cd, export, history, type, test, and alias.",
        not_this="Not a shell inside bhuppi. Not eval of a generated string.",
        section="Shells",
        complete="none",
    )
    c.add("lang/bash/run", "run a script", "Run a script with bash.", bin="bash", special="args-bin", shape="bhuppi lang bash run SCRIPT", example="bhuppi lang bash run script.sh", argv=["bash"], user_args="append", complete="files", min_args=1, runs="bash script.sh")
    c.add("lang/bash/check", "syntax check", "Parse a script with bash -n. Nothing is executed.", bin="bash", special="args-bin", shape="bhuppi lang bash check SCRIPT", example="bhuppi lang bash check script.sh", argv=["bash", "-n"], user_args="append", complete="files", min_args=1, runs="bash -n script.sh", rule="The argv is bash -n and the files you name. bash -n does not run the script.")
    c.add("lang/bash/flags", "bash with your flags", "Run bash with the flags you type.", bin="bash", shape="bhuppi lang bash flags ARGS", example="bhuppi lang bash flags -lc 'printf hi'", argv=["bash"], complete="none", min_args=1, desc="Arguments are passed to bash as separate argv elements. Your shell quotes them before bhuppi sees them. --dry-run prints bash and those arguments.")
    c.add(
        "lang/bash/builtin",
        "shell builtins",
        "cd, export, history, type, test, and alias are shell builtins. bhuppi teaches them and does not inject them, except the files go / folders go hook for cd.",
        delivery="skip",
        shape="bhuppi lang bash builtin",
        example="bhuppi lang bash builtin",
        when="You looked up a builtin and want to know why there is no wrapper.",
        rule="A child process cannot change the parent shell. The go hook is the one exception, and it is explicit.",
        desc="explain export, alias, history, type, test, source, eval, jobs, bg, and fg land here or on a skip entry. files go and folders go are the cd hook.",
        not_this="Not `source` of an rc file. Not eval.",
        complete="none",
        user_args="none",
    )
    for name, title, bin_name, unix, what, example in [
        ("sh", "POSIX sh", "sh", "sh dash", "Run a script with sh. On macOS sh is often bash or dash in POSIX mode; bhuppi execs whatever sh is on PATH.", "bhuppi lang sh script.sh"),
        ("zsh", "zsh", "zsh", "zsh", "Run a script with zsh.", "bhuppi lang zsh script.zsh"),
        ("fish", "fish", "fish", "fish", "Run a script with fish, when fish is installed.", "bhuppi lang fish script.fish"),
        ("perl", "perl", "perl", "perl", "Run perl with the arguments you type.", "bhuppi lang perl script.pl"),
        ("python", "python", "python3", "python python3", "Run python3. The name python is the same leaf.", "bhuppi lang python script.py"),
        ("ruby", "ruby", "ruby", "ruby", "Run ruby with the arguments you type.", "bhuppi lang ruby script.rb"),
        ("printf", "printf", "printf", "printf", "Format a line by running printf, not the shell builtin.", "bhuppi lang printf '%s\\n' hi"),
        ("echo", "echo", "echo", "echo", "Print arguments by running the echo binary, not the shell builtin.", "bhuppi lang echo hi"),
        ("time", "time a command", "time", "time", "Time a command by running /usr/bin/time when that binary is first on PATH. The shell keyword time is not wrapped.", "bhuppi lang time ls"),
    ]:
        kw = dict(bin=bin_name, unix=unix.split(), shape=f"bhuppi lang {name} [ARGS]", example=example, section="Interpreters", complete="files", min_args=0 if name in ("echo", "printf") else 1)
        if name == "python":
            kw["bin"] = "python3"
            kw["prefer"] = ["python3", "python"]
        if name == "fish":
            kw["optional"] = True
            kw["hint"] = "brew install fish    or    apt install fish"
        c.add(f"lang/{name}", title, what, **kw)


def proc(c: Cat) -> None:
    c.add(
        "proc/list",
        "list processes",
        "List processes as numbered boxes. A keyword keeps the processes whose command line contains that word.",
        bin="ps",
        unix=["ps"],
        special="procs",
        shape="bhuppi proc list [KEYWORD]",
        example="bhuppi proc list ssh",
        examples=["bhuppi proc list", "bhuppi proc list -ef"],
        when="You want every process, or the ones whose command matches a keyword, in a short box instead of a raw ps line.",
        rule="No arguments, or one keyword, runs ps and bhuppi draws a box for each match. --dry-run prints that ps argv and says bhuppi draws the boxes. The command line in a box is the program path plus a short role. Flags you type are passed to ps and are not drawn as boxes.",
        desc="Each box has pid, user, state, nice, priority, cpu, memory, elapsed, program, command, called by, children, and working directory. A long argument string is not printed.",
        not_this="Not a process killer. Not top. Not ps -C. Not a raw ps aux dump.",
        runs="ps -axww -o pid=,ppid=,user=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=",
        runs_linux="ps -eww -o pid=,ppid=,user=,ni=,pri=,stat=,%cpu=,%mem=,etime=,args=",
        also_linux="bhuppi proc list ssh",
        also_macos="bhuppi proc list ssh",
        sparkle=True,
        section="Look",
        complete="none",
        user_args="none",
        min_args=0,
    )
    c.add(
        "proc/stat",
        "process properties",
        "Show one process: pid, user, uid, nice, priority, state, cpu, memory, elapsed, and command.",
        bin="ps",
        unix=[],
        special="pstat",
        shape="bhuppi proc stat PID",
        example="bhuppi proc stat 123",
        when="You want the properties of one pid, including its nice value, the way files stat shows a file.",
        rule="macOS runs ps -p PID -o pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=. The ni column is the nice value. Linux runs ps -p PID -o pid,user,uid,ni,pri,stat,%cpu,%mem,etime,args. --os changes the help examples only. A flag you type is not rewritten.",
        desc="Fields: pid, user, uid, nice, priority, state, cpu, memory, elapsed, command. --dry-run prints the argv for this OS.",
        not_this="Not files stat. Not ls. Not a GNU ps -o layout forced onto macOS.",
        runs="ps -p 123 -o pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=",
        runs_linux="ps -p 123 -o pid,user,uid,ni,pri,stat,%cpu,%mem,etime,args",
        runs_darwin="ps -p 123 -o pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=",
        also_linux="ps -p 123 -o pid,user,uid,ni,pri,stat,%cpu,%mem,etime,args",
        also_macos="ps -p 123 -o pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=",
        section="Look",
        complete="none",
        min_args=1,
        user_args="none",
    )
    c.add(
        "proc/using",
        "who is using a command, path, or port",
        "Show who holds a command, a path, or a port by running lsof.",
        bin="lsof",
        unix=["lsof"],
        special="using",
        shape="bhuppi proc using KEYWORD | PATH | :PORT",
        example="bhuppi proc using python",
        examples=["bhuppi proc using ~/.zshrc", "bhuppi proc using :443"],
        when="You have a command name, a file path, or a port and want the processes that have it open.",
        rule="A bare word with no slash is lsof -c KEYWORD (the command name begins with KEYWORD). A path, one that contains / or names an existing file, is lsof -- PATH. :PORT is lsof -i :PORT. A bare number is a port. Flags you type yourself are passed through.",
        desc="Three shapes: keyword, path, and port. --dry-run prints the lsof argv. This is an admin view of your own machine.",
        not_this="A bare word is not a filename. Not lsof WORD. Not a port scanner. Not nmap. Not a stealth scan.",
        runs="lsof -c python",
        also_linux="lsof -c python",
        also_macos="lsof -- ~/.zshrc",
        sparkle=True,
        section="Look",
        complete="none",
        min_args=0,
        user_args="none",
        hint="lsof ships with macOS. On Linux: apt install lsof",
    )
    c.add(
        "proc/view",
        "one process",
        "Show one process on a single page: the program file, the programs that called it, its children, its working directory, and its connections.",
        bin="ps",
        unix=[],
        special="pview",
        shape="bhuppi proc view PID|KEYWORD",
        example="bhuppi proc view 1",
        when="You want one readable snapshot of a process you named, not a live screen.",
        rule="A pid is shown directly. A keyword is pgrep -lf KEYWORD. One match opens that pid. Several matches list the pids and stop. --dry-run prints the ps and lsof argv, one per line. This is one shot. It does not loop and it does not run top.",
        desc="Sections: process, program file, called by, children, working directory, connections. Numbers are cyan. The program path is yellow.",
        not_this="Not strace. Not dtruss. Not a stealth tracer. Not top.",
        section="Look",
        complete="none",
        min_args=1,
        user_args="none",
    )
    c.add(
        "proc/agents",
        "local agents",
        "Show local AI agent processes on this machine: pid, user, name, and program path.",
        bin="ps",
        unix=[],
        special="pagents",
        shape="bhuppi proc agents [NAME|PID]",
        example="bhuppi proc agents",
        examples=["bhuppi proc agents cursor"],
        when="You want to see which local agents are running on this machine.",
        rule="Matching is local and case-insensitive against cursor, claude, codex, antigravity, deepseek, qwen, chatgpt, copilot, windsurf, and ollama. A name shows up to 8 processes. More than 8 lists pids. A sensitive path is marked and is not opened. File contents and environment variables are not printed.",
        desc="No name prints a short table. A name or a pid prints one concise page, the same facts as proc view, plus open paths and outbound remotes. A bare address or a port other than 80 or 443 is marked. Normal HTTPS stays unmarked.",
        not_this="Not a secret dump. Not dtruss. Not a remote scan. Not a network sweep.",
        section="Look",
        complete="none",
        min_args=0,
        user_args="none",
    )
    c.add(
        "proc/top",
        "live process view",
        "Open top. htop is a separate name in explain when you want that binary.",
        bin="top",
        unix=["top", "htop"],
        shape="bhuppi proc top [ARGS]",
        example="bhuppi proc top",
        when="You want the live view.",
        rule="The argv is top plus flags you type. The live program keeps the terminal. A one-shot snapshot is a flag you type: top -l 1 on macOS, top -b -n 1 on Linux. bhuppi does not translate one into the other.",
        desc="top is interactive and is exec'd with the tty. Pass the flags you want.",
        not_this="Not a sampler that rewrites flags between operating systems. Not GNU top -b -n 1 on macOS.",
        also_linux="top -b -n 1",
        also_macos="top -l 1",
        runs="top",
        section="Look",
        complete="none",
        user_args="append",
        argv=["top"],
    )
    for name, title, bin_name, unix, what, shape, example, min_args, extra in [
        ("kill", "signal a process", "kill", "kill", "Signal a pid by running kill.", "bhuppi proc kill PID", "bhuppi proc kill 123", 1, {"rule": "The argv is kill plus what you typed. -9 is not added unless you typed it.", "not_this": "Not kill -9 by default. Not pkill."}),
        ("pkill", "signal by name", "pkill", "pkill", "Signal processes by name by running pkill.", "bhuppi proc pkill NAME", "bhuppi proc pkill sleep", 1, {}),
        ("pgrep", "find pids by name", "pgrep", "pgrep", "Print pids by name by running pgrep.", "bhuppi proc pgrep NAME", "bhuppi proc pgrep ssh", 1, {}),
        ("killall", "signal by name", "killall", "killall", "Signal by name by running killall. On macOS this is not the Linux killall.", "bhuppi proc killall NAME", "bhuppi proc killall sleep", 1, {"also_linux": "killall matches the process name", "also_macos": "macOS killall matches the application or process name; read the local man page before using it"}),
        ("nice", "current niceness or start with nice", "nice", "nice", "Print the current niceness, or run a command with nice.", "bhuppi proc nice    |    bhuppi proc nice -n N CMD", "bhuppi proc nice", 0, {
            "special": "nice",
            "examples": ["bhuppi proc nice -n 10 ls"],
            "when": "You want the current niceness, or you want to start a command at a niceness you choose.",
            "rule": "A command is passed through unchanged, including -n. bhuppi does not swallow -n. With no command, Linux runs nice, which prints the current niceness. macOS /usr/bin/nice with no utility prints usage, so this page runs ps -o ni= -p of this process. That value is the nice column.",
            "desc": "Both shapes are valid: bhuppi proc nice, and bhuppi proc nice -n N CMD. --dry-run prints the argv for this OS.",
            "not_this": "Not renice of a running pid. Not a rewritten -n. Not the macOS nice usage text.",
            "runs": "ps -o ni= -p PID",
            "runs_linux": "nice",
            "runs_darwin": "ps -o ni= -p PID",
            "also_linux": "nice",
            "also_macos": "ps -o ni= -p PID",
            "user_args": "none",
        }),
        ("renice", "change niceness", "renice", "renice", "Change niceness of a running pid by running renice.", "bhuppi proc renice INCREMENT PID", "bhuppi proc renice 10 123", 2, {
            "examples": ["bhuppi proc renice -n 10 -p 123"],
            "when": "You want to change the niceness of a pid that is already running.",
            "rule": "The argv is renice plus the arguments you typed. macOS accepts INCREMENT PID and also renice -n INCREMENT -p PID. Flags you type are not rewritten, and a GNU-only layout is not required.",
            "desc": "The example is the macOS form renice INCREMENT PID. --dry-run prints that argv.",
            "not_this": "Not a GNU-only flag layout. Not proc nice, which starts a new command.",
            "runs": "renice 10 123",
            "also_macos": "renice -n 10 -p 123",
            "also_linux": "renice -n 10 -p 123",
        }),
        ("nohup", "survive hangup", "nohup", "nohup", "Run a command immune to hangup by running nohup.", "bhuppi proc nohup CMD", "bhuppi proc nohup sleep 2", 1, {}),
        ("pstree", "process tree", "ps", "pstree", "Draw a process tree from ps. Same picture as proc map. The pstree binary is not required.", "bhuppi proc pstree [--depth N]", "bhuppi proc pstree", 0, {
            "special": "pmap",
            "user_args": "none",
            "examples": ["bhuppi proc pstree --depth 2"],
            "when": "You want a process tree on this machine. pstree the binary is not required.",
            "rule": "Discovery is ps. macOS runs ps -axo pid=,ppid=,user=,ni=,stat=,command=. Linux runs ps -eo pid=,ppid=,user=,ni=,stat=,args=. bhuppi draws the box-drawing tree from that table. Default depth is 3. --depth N sets it. --dry-run prints the ps argv and says the drawing is bhuppi's tree view. A long command is trimmed so the level tag stays visible.",
            "desc": "proc pstree and proc map draw the same tree. This is the local process list, not a scan of other machines.",
            "not_this": "Not a stealth scan. Not raw pstree text. Not an installer.",
            "runs": "ps -axo pid=,ppid=,user=,ni=,stat=,command=",
            "runs_linux": "ps -eo pid=,ppid=,user=,ni=,stat=,args=",
            "runs_darwin": "ps -axo pid=,ppid=,user=,ni=,stat=,command=",
            "also_linux": "ps -eo pid=,ppid=,user=,ni=,stat=,args=",
            "also_macos": "ps -axo pid=,ppid=,user=,ni=,stat=,command=",
        }),
        ("strace", "trace syscalls", "strace", "strace", "Trace syscalls of a command you name by running strace. Linux only.", "bhuppi proc strace CMD", "bhuppi proc strace ls", 1, {
            "platform": "linux",
            "fallback": "proc strace",
            "docs": "https://developer.apple.com/library/archive/documentation/Darwin/Conceptual/KernelProgramming/build/build.html",
            "hint": "strace is Linux-only. On macOS, dtruss is docs-only: System Integrity Protection blocks it for ordinary use. bhuppi does not run it and does not list flags. See: bhuppi help proc strace",
            "not_this": "Not dtruss. Not a stealth tracer.",
            "when": "You are on Linux and want a syscall trace of a command you name.",
            "rule": "On Linux the argv is strace plus the command you typed. On macOS bhuppi exits 2 and names strace before it asks for a command. dtruss is docs-only on that page: name, purpose, and the official docs link. No flag recipe and no exec.",
            "desc": "Linux-only. The darwin note on this page is documentation for dtruss under SIP. Nothing is traced on macOS.",
            "runs": "strace ls",
            "also_linux": "strace ls",
            "also_macos": "dtruss is docs-only on macOS (SIP). Not executed.",
        }),
    ]:
        c.add(f"proc/{name}", title, what, bin=bin_name, unix=[unix] if isinstance(unix, str) else unix, shape=shape, example=example, min_args=min_args, section="Signals", complete="none", **extra)
    c.add(
        "proc/map",
        "process tree",
        "Draw a process tree from ps. The picture is bhuppi's, with box drawing and a level on each process under the roots.",
        bin="ps",
        unix=[],
        special="pmap",
        shape="bhuppi proc map [--depth N]",
        example="bhuppi proc map --depth 2",
        examples=["bhuppi proc pstree --depth 2"],
        when="You want a process tree on this machine, including when pstree is not installed.",
        rule="Discovery is ps. macOS runs ps -axo pid=,ppid=,user=,ni=,stat=,command=. Linux runs ps -eo pid=,ppid=,user=,ni=,stat=,args=. bhuppi draws the box-drawing tree from that table. Default depth is 3. --depth N sets it. --dry-run prints the ps argv and says the drawing is bhuppi's tree view. A long command is trimmed so the level tag stays visible. proc pstree draws this same tree.",
        desc="pid, user, and the command are one row. This is the local process list, not a scan of other machines.",
        not_this="Not a stealth scan. Not the pstree binary. Not a flat ps listing.",
        runs="ps -axo pid=,ppid=,user=,ni=,stat=,command=",
        runs_linux="ps -eo pid=,ppid=,user=,ni=,stat=,args=",
        runs_darwin="ps -axo pid=,ppid=,user=,ni=,stat=,command=",
        also_linux="ps -eo pid=,ppid=,user=,ni=,stat=,args=",
        also_macos="ps -axo pid=,ppid=,user=,ni=,stat=,command=",
        section="Look",
        complete="none",
        min_args=0,
        user_args="none",
    )


def net(c: Cat) -> None:
    c.add("net/get", "fetch a URL", "Fetch a URL by running curl.", bin="curl", unix=["curl"], shape="bhuppi net get URL", example="bhuppi net get https://example.com", when="You want the body of a URL you already trust, on stdout.", rule="The argv is curl and the arguments you typed. -L is not added unless you type it.", desc="URL is the first argument. Further curl flags are yours and appear in --dry-run.", not_this="Not a crawler. Not wget's recursive mirror.", runs="curl https://example.com", section="Reach", complete="none", min_args=1, sparkle=True)
    c.add("net/wget", "fetch a URL with wget", "Fetch one URL by running wget, when wget is installed.", bin="wget", unix=["wget"], shape="bhuppi net wget URL", example="bhuppi net wget https://example.com/file", when="curl is not the tool you want, and the URL is one you already trust.", rule="The argv is wget and your arguments. Recursive mirroring is not the default.", desc="One URL is the ordinary shape. Flags you type are passed through.", not_this="Not a way to pull untrusted payloads. Not a recursive site mirror unless you typed those flags yourself.", hint="brew install wget    or    apt install wget", optional=True, section="Reach", complete="none", min_args=1)
    c.add("net/ping", "ping a host", "Ping a host four times.", bin="ping", unix=["ping"], special="ping", shape="bhuppi net ping HOST", example="bhuppi net ping example.com", when="You want a short reachability check that ends.", rule="The default argv is ping -c 4 HOST on Unix and ping -n 4 HOST on Windows, and --dry-run prints that. If your first argument is already a flag, bhuppi does not add another count.", desc="HOST is required. This is ICMP to a host you named, not a sweep.", not_this="Not a network sweep. Not nmap.", runs="ping -c 4 example.com", runs_windows="ping -n 4 example.com", also_linux="ping -c 4 example.com", also_macos="ping -c 4 example.com", section="Reach", complete="none", min_args=1)
    c.add(
        "net/dns",
        "resolve a name",
        "Resolve a name with dig when it exists, otherwise host.",
        bin="dig",
        prefer=["dig", "host"],
        unix=["dig", "host", "nslookup"],
        special="dns",
        shape="bhuppi net dns NAME",
        example="bhuppi net dns example.com",
        when="You want an ordinary lookup for a name you operate or you are debugging.",
        rule="AXFR is refused. The argv is dig NAME or host NAME. No zone transfer, no +bufsize games.",
        desc="The Unix command host(1) lives here. The machine group is `bhuppi host`, which is uname and packages, not this lookup.",
        not_this="Not a zone transfer. Not a subdomain brute force.",
        runs="dig example.com",
        also_macos="host example.com",
        section="Reach",
        complete="none",
        min_args=1,
        explain_line="net dns — dig or host. The machine group is bhuppi host, not this lookup.",
    )
    c.add(
        "net/ssh",
        "ssh",
        "Connect, make a key, or copy with scp. Each child execs the real binary.",
        bin="ssh",
        unix=["ssh"],
        delivery="language-mode",
        shape="bhuppi net ssh connect|key|copy …",
        example="bhuppi net ssh connect user@host",
        when="You operate the host you are logging into.",
        rule="connect is ssh. key is ssh-keygen. copy is scp. No proxy jump is added unless you type it.",
        desc="Three children. Bare `bhuppi net ssh` prints this page.",
        not_this="Not a tunnel factory. Not an unattended backdoor.",
        sparkle=True,
        section="Reach",
        complete="none",
    )
    c.add("net/ssh/connect", "open a session", "Open an ssh session.", bin="ssh", shape="bhuppi net ssh connect USER@HOST", example="bhuppi net ssh connect user@host", runs="ssh user@host", min_args=1, complete="none", argv=["ssh"])
    c.add("net/ssh/key", "make a key", "Create an ed25519 key with ssh-keygen.", bin="ssh-keygen", unix=["ssh-keygen"], shape="bhuppi net ssh key [FILE]", example="bhuppi net ssh key ~/.ssh/id_ed25519", argv=["ssh-keygen", "-t", "ed25519"], user_args="default", runs="ssh-keygen -t ed25519", desc="With no arguments the argv is ssh-keygen -t ed25519. If you pass arguments, they replace that default and are not merged.", not_this="Not a key harvester. The file you name is the one ssh-keygen writes.", complete="files", min_args=0)
    c.add("net/ssh/copy", "copy over ssh", "Copy files with scp.", bin="scp", unix=["scp", "sftp"], shape="bhuppi net ssh copy SRC DST", example="bhuppi net ssh copy file user@host:", runs="scp file user@host:", min_args=2, complete="both", argv=["scp"], desc="The argv is scp plus your arguments. sftp is the sibling binary; explain sftp points here. bhuppi does not rewrite scp into sftp.")
    c.add(
        "net/ports",
        "listening ports",
        "Show listeners. Linux prefers ss; otherwise netstat. macOS uses netstat.",
        bin="ss",
        special="ports",
        unix=["ss", "netstat"],
        shape="bhuppi net ports",
        example="bhuppi net ports",
        when="You want to see what is listening on this machine.",
        rule="The binary is chosen for this OS and printed by --dry-run. ss is not rewritten into a BSD-shaped ss.",
        desc="Linux: ss -lptn when ss exists, else netstat -lptn. macOS: netstat -an. This is local state, not a scan of other hosts.",
        not_this="Not nmap. Not a sweep.",
        runs_linux="ss -lptn",
        runs_darwin="netstat -an",
        also_linux="ss -lptn",
        also_macos="netstat -an",
        sparkle=True,
        section="Look",
        complete="none",
        user_args="none",
        min_args=0,
    )
    c.add(
        "net/connections",
        "inbound and outbound",
        "Show inbound and outbound connections on this machine.",
        bin="lsof",
        special="conns",
        shape="bhuppi net connections",
        example="bhuppi net connections",
        when="You want every connection on this machine, not only the listeners.",
        rule="The argv is lsof -n -P -i. --dry-run prints that argv and one line that bhuppi then resolves names. It does not contact VirusTotal. Listening sockets are inbound. Other connections are outbound, grouped by host and port. The numeric address stays, and a reverse-DNS name is shown beside it when one exists. A bare address with no name, or a port other than 80 or 443, is marked. reputation-check is separate. It is the only command that contacts VirusTotal and AbuseIPDB. Keys come from bhuppi config reputation set vt and bhuppi config reputation set abuseip.",
        desc="This is local state from lsof. bhuppi then resolves names for public addresses. Loopback and private addresses are not sent to a resolver or to either reputation service.",
        not_this="Not nmap. Not a remote scan. Reputation-check is not the default. Keys are not command arguments. Private addresses are not sent to either service.",
        examples=["bhuppi net connections reputation-check", "bhuppi config reputation set vt", "bhuppi config reputation set abuseip"],
        more="bhuppi config reputation set vt · bhuppi config reputation set abuseip · bhuppi config reputation status",
        runs="lsof -n -P -i",
        section="Look",
        complete="none",
        user_args="none",
        min_args=0,
        sparkle=True,
    )
    c.add(
        "net/ip",
        "addresses",
        "Show addresses. Linux prefers ip addr, else ifconfig. macOS uses ifconfig.",
        bin="ip",
        special="ipaddr",
        unix=["ip", "ifconfig"],
        shape="bhuppi net ip",
        example="bhuppi net ip",
        when="You want this machine's interface addresses.",
        rule="ip is Linux-only as a required tool. On macOS the argv is ifconfig and the missing-ip error points here.",
        desc="--dry-run prints ip addr or ifconfig. Flags are not translated between them.",
        not_this="Not a route injector.",
        runs_linux="ip addr",
        runs_darwin="ifconfig",
        also_linux="ip addr",
        also_macos="ifconfig",
        platform="any",
        section="Look",
        complete="none",
        user_args="none",
    )
    c.add(
        "net/filter",
        "list firewall rules",
        "List firewall rules. One child per tool. List only.",
        delivery="capability",
        shape="bhuppi net filter iptables|nft|ufw|pfctl",
        example="bhuppi net filter pfctl",
        when="You want to read rules on a host you administer.",
        rule="Each child runs a fixed list command. Extra flags are refused. Nothing is inserted, deleted, or flushed.",
        desc="Linux: iptables, nft, ufw. macOS: pfctl. A missing Linux tool on macOS exits 2 and names the pfctl page.",
        not_this="Not a rule writer. Not a bypass.",
        section="Look",
        complete="none",
    )
    c.add("net/filter/iptables", "list iptables", "List iptables rules.", bin="iptables", unix=["iptables"], special="fixed", argv=["iptables", "-L", "-n"], platform="linux", fallback="net filter pfctl", hint="iptables is Linux-only. On macOS: bhuppi help net filter pfctl", shape="bhuppi net filter iptables", example="bhuppi net filter iptables", runs="iptables -L -n", user_args="none", complete="none", not_this="Not iptables -F, -A, or -D. List only.")
    c.add("net/filter/nft", "list nftables", "List the nftables ruleset.", bin="nft", unix=["nft"], special="fixed", argv=["nft", "list", "ruleset"], platform="linux", fallback="net filter pfctl", hint="nft is Linux-only. On macOS: bhuppi help net filter pfctl", shape="bhuppi net filter nft", example="bhuppi net filter nft", runs="nft list ruleset", user_args="none", complete="none", not_this="Not nft add or delete. List only.")
    c.add("net/filter/ufw", "list ufw", "Show ufw status.", bin="ufw", unix=["ufw"], special="fixed", argv=["ufw", "status"], platform="linux", fallback="net filter pfctl", hint="ufw is Linux-only. On macOS: bhuppi help net filter pfctl", shape="bhuppi net filter ufw", example="bhuppi net filter ufw", runs="ufw status", user_args="none", complete="none", not_this="Not ufw allow or deny. List only.")
    c.add("net/filter/pfctl", "list pf rules", "List pf rules.", bin="pfctl", unix=["pfctl"], special="fixed", argv=["pfctl", "-sr"], platform="darwin", fallback="net filter iptables", hint="pfctl is macOS. On Linux: bhuppi help net filter iptables", shape="bhuppi net filter pfctl", example="bhuppi net filter pfctl", runs="pfctl -sr", user_args="none", complete="none", not_this="Not pfctl -f of a ruleset bhuppi wrote. List only.")
    c.add("net/traceroute", "trace a path", "Trace a path by running traceroute.", bin="traceroute", unix=["traceroute", "tracepath"], shape="bhuppi net traceroute HOST", example="bhuppi net traceroute example.com", min_args=1, complete="none", section="Reach", not_this="Not a stealth scan.")
    c.add(
        "net/tcpdump",
        "capture on an interface you admin",
        "Run tcpdump -i IFACE -n, with an optional packet count.",
        bin="tcpdump",
        unix=["tcpdump"],
        special="tcpdump",
        shape="bhuppi net tcpdump IFACE [COUNT]",
        example="bhuppi net tcpdump en0 20",
        when="You administer the interface and want a short, visible capture.",
        rule="The argv is tcpdump -i IFACE -n and, if you pass COUNT, -c COUNT. Other flags are refused.",
        desc="COUNT limits the capture so it ends. There is no stealth mode and no middlebox mode.",
        not_this="Not a stealth capture. Not a MITM. Not a promiscuous setup beyond what tcpdump -i does on an interface you named.",
        runs="tcpdump -i en0 -n -c 20",
        section="Look",
        complete="none",
        min_args=1,
        user_args="none",
    )
    c.add("net/arp", "ARP table", "Show the ARP table by running arp.", bin="arp", unix=["arp"], shape="bhuppi net arp", example="bhuppi net arp -a", user_args="default", argv=["arp", "-a"], complete="none", not_this="Not ARP spoofing.")
    c.add("net/route", "routing table", "Show the routing table by running route.", bin="route", unix=["route"], special="route", shape="bhuppi net route", example="bhuppi net route", complete="none", user_args="none", not_this="Not a route change. The default is the show command for this OS.", runs_linux="ip route", runs_darwin="route -n get default", desc="On Linux, if ip exists, the argv is ip route; otherwise route -n. On macOS the argv is route -n get default. --dry-run prints which. This page does not add or delete routes.")
    c.add(
        "net/nmap",
        "nmap is docs-only",
        "nmap is a port scanner. bhuppi names it and does not run it.",
        bin="nmap",
        unix=["nmap"],
        delivery="docs-only",
        shape="bhuppi net nmap",
        example="bhuppi explain nmap",
        when="You want the official manual, not a scan from this atlas.",
        rule="No flags are accepted and nmap is not executed.",
        desc="Read the official book for the tool. bhuppi's job stops at the name, the purpose, and the link.",
        not_this="Not a scan. Not a version probe. Not a host discovery run.",
        docs="https://nmap.org/book/man.html",
        complete="none",
        user_args="none",
    )
    c.add(
        "net/nc",
        "netcat is docs-only",
        "nc and netcat open TCP connections and listeners. bhuppi names them and does not run them.",
        bin="nc",
        unix=["nc", "netcat", "ncat"],
        delivery="docs-only",
        shape="bhuppi net nc",
        example="bhuppi explain nc",
        when="You want the official manual.",
        rule="No flags are accepted and nc is not executed.",
        desc="There is no listener recipe and no payload recipe on this page.",
        not_this="Not a bind shell. Not a reverse shell. Not a banner grabber wrapped by bhuppi.",
        docs="https://man7.org/linux/man-pages/man1/ncat.1.html",
        complete="none",
        user_args="none",
    )


def pack(c: Cat) -> None:
    c.add(
        "pack/tar",
        "tar archives",
        "Create, extract, list, or gzip a tar archive.",
        bin="tar",
        unix=["tar"],
        delivery="language-mode",
        shape="bhuppi pack tar create|extract|list|compress …",
        example="bhuppi pack tar compress logs.tgz logs",
        when="You are packing or opening a tar archive.",
        rule="Each child is a fixed tar mode. Exclude-flag spelling is documented, not rewritten.",
        desc="create is -cf, extract is -xf, list is -tf, compress is -czf. --dry-run prints that argv.",
        not_this="Not a second tar. Not an auto-detected mode.",
        also_linux="tar --exclude=PATTERN is the GNU spelling",
        also_macos="tar --exclude PATTERN is the BSD spelling you type yourself",
        section="Archives",
    )
    c.add("pack/tar/create", "create an archive", "Create an uncompressed tar.", bin="tar", shape="bhuppi pack tar create ARCHIVE PATHS", example="bhuppi pack tar create logs.tar logs", argv=["tar", "-cf"], runs="tar -cf logs.tar logs", min_args=2, complete="both", complete_skip=1)
    c.add("pack/tar/extract", "extract an archive", "Extract a tar archive.", bin="tar", shape="bhuppi pack tar extract ARCHIVE", example="bhuppi pack tar extract logs.tar", argv=["tar", "-xf"], runs="tar -xf logs.tar", min_args=1, complete="files")
    c.add("pack/tar/list", "list an archive", "List the contents of a tar archive.", bin="tar", shape="bhuppi pack tar list ARCHIVE", example="bhuppi pack tar list logs.tar", argv=["tar", "-tf"], runs="tar -tf logs.tar", min_args=1, complete="files")
    c.add("pack/tar/compress", "gzip an archive", "Create a gzip-compressed tar.", bin="tar", shape="bhuppi pack tar compress ARCHIVE PATHS", example="bhuppi pack tar compress logs.tgz logs", argv=["tar", "-czf"], runs="tar -czf logs.tgz logs", min_args=2, complete="both", complete_skip=1)
    for name, title, bin_name, unix, what, example in [
        ("gzip", "gzip", "gzip", "gzip gunzip zcat", "Compress or pass flags through to gzip. gunzip and zcat are the same family.", "bhuppi pack gzip file"),
        ("bzip2", "bzip2", "bzip2", "bzip2 bunzip2", "Compress with bzip2.", "bhuppi pack bzip2 file"),
        ("xz", "xz", "xz", "xz", "Compress with xz.", "bhuppi pack xz file"),
        ("zip", "zip", "zip", "zip", "Create a zip archive.", "bhuppi pack zip out.zip files"),
        ("unzip", "unzip", "unzip", "unzip", "List or extract a zip archive. Pass -l yourself to list.", "bhuppi pack unzip out.zip"),
        ("cpio", "cpio", "cpio", "cpio", "Run cpio with the arguments you type.", "bhuppi pack cpio -t < archive.cpio"),
        ("zstd", "zstd", "zstd", "zstd", "Compress with zstd, when it is installed.", "bhuppi pack zstd file"),
    ]:
        kw = dict(bin=bin_name, unix=unix.split(), shape=f"bhuppi pack {name} [ARGS]", example=example, section="Compress", complete="files", min_args=1)
        if name == "zstd":
            kw["optional"] = True
            kw["hint"] = "brew install zstd    or    apt install zstd"
        c.add(f"pack/{name}", title, what, **kw)


def disk(c: Cat) -> None:
    c.add(
        "disk/review",
        "what filled the disk",
        "Show the volume, the largest folders in your home directory, and what changed.",
        bin="df",
        special="dreview",
        shape="bhuppi disk review [--hour|--day|--week|--month|--since Nh|Nd|Nw]",
        example="bhuppi disk review",
        when="The disk is full and macOS free space is hard to read.",
        rule="Default prints the disk summary, all-time top folders, and the last hour, day, week, and month. One flag prints that interval and the summary, not the other intervals. On macOS the summary is the APFS container from diskutil apfs list, and each volume in that container is listed under it. On Linux the summary is df of /. --dry-run prints that argv and does not walk the disk. Changes are scoped to the home directory. Purgeable space stays separate from free.",
        desc="Safe to remove lists rebuildable caches and does not delete them. bhuppi disk clean is the next command, and it does nothing until --yes.",
        not_this="Not a wipe. Not rm -rf of the home directory. bhuppi disk clean without --yes deletes nothing.",
        section="Look",
        complete="none",
        user_args="none",
        min_args=0,
        sparkle=True,
    )
    c.add(
        "disk/clean",
        "remove rebuildable caches",
        "Delete the contents of known rebuildable cache directories when you add --yes.",
        special="dclean",
        shape="bhuppi disk clean [--yes]",
        example="bhuppi disk clean",
        when="disk review showed caches you are willing to let apps recreate.",
        rule="Without --yes the page lists the allowlist and deletes nothing. --yes deletes the contents of those directories, not the directories, and nothing outside them. Each path is printed before it is removed. --dry-run prints the argv and does not delete.",
        desc="The allowlist is ~/Library/Caches, ~/Library/Logs, ~/.npm/_cacache, and ~/Library/Caches/Homebrew. A missing path is skipped.",
        not_this="Not a wipe. Not rm -rf of the home directory. bhuppi disk clean without --yes deletes nothing.",
        section="Change",
        complete="none",
        user_args="none",
        min_args=0,
    )
    c.add("disk/space", "free space", "Show filesystem free space by running df -h.", bin="df", unix=["df"], shape="bhuppi disk space [PATH]", example="bhuppi disk space", argv=["df", "-h"], runs="df -h", section="Look", complete="folders", user_args="append")
    c.add("disk/usage", "directory size", "Show disk usage by running du -h.", bin="du", unix=["du"], shape="bhuppi disk usage [PATH]", example="bhuppi disk usage .", argv=["du", "-h"], runs="du -h .", desc="The argv starts with du -h and then your paths. Depth flags differ by OS and are not rewritten; the disk-hogs recipe prints an OS-honest pipeline.", also_linux="du -h --max-depth=1 .", also_macos="du -h -d 1 .", section="Look", complete="folders", user_args="append")
    c.add(
        "disk/diskutil",
        "diskutil",
        "list, info, or verify a disk with diskutil. macOS.",
        bin="diskutil",
        unix=["diskutil"],
        delivery="capability",
        platform="darwin",
        fallback="disk lsblk",
        hint="diskutil is macOS. On Linux: bhuppi help disk lsblk",
        shape="bhuppi disk diskutil list|info|verify",
        example="bhuppi disk diskutil list",
        when="You want a read or a verify of a disk on this Mac.",
        rule="Only list, info, and verify. Erase and partition are not children.",
        desc="On Linux this group points at lsblk. --dry-run prints diskutil and the subcommand.",
        not_this="Not diskutil eraseDisk. Not a partition writer.",
        section="Volumes",
    )
    c.add("disk/diskutil/list", "list disks", "List disks.", bin="diskutil", special="fixed", argv=["diskutil", "list"], shape="bhuppi disk diskutil list", example="bhuppi disk diskutil list", runs="diskutil list", platform="darwin", fallback="disk lsblk", user_args="none", complete="none")
    c.add("disk/diskutil/info", "disk info", "Show one disk.", bin="diskutil", shape="bhuppi disk diskutil info DISK", example="bhuppi disk diskutil info disk0", argv=["diskutil", "info"], runs="diskutil info disk0", platform="darwin", fallback="disk lsblk", min_args=1, complete="none")
    c.add("disk/diskutil/verify", "verify a volume", "Verify a volume.", bin="diskutil", shape="bhuppi disk diskutil verify DISK", example="bhuppi disk diskutil verify disk0s2", argv=["diskutil", "verifyVolume"], runs="diskutil verifyVolume disk0s2", platform="darwin", fallback="disk lsblk", min_args=1, complete="none", not_this="Not repair. verifyVolume only.")
    c.add("disk/mount", "mount table", "Show mounts by running mount.", bin="mount", unix=["mount"], shape="bhuppi disk mount", example="bhuppi disk mount", argv=["mount"], user_args="none", special="fixed", not_this="Not a request to mount a new filesystem. This page only prints the table.", runs="mount", section="Volumes", complete="none")
    c.add("disk/umount", "unmount", "Unmount a path you name by running umount.", bin="umount", unix=["umount"], shape="bhuppi disk umount PATH", example="bhuppi disk umount /mnt/data", min_args=1, complete="folders", not_this="Not an unmount of a path you did not type.", section="Volumes")
    c.add("disk/lsblk", "list block devices", "List block devices by running lsblk. Linux.", bin="lsblk", unix=["lsblk"], shape="bhuppi disk lsblk", example="bhuppi disk lsblk", platform="linux", fallback="disk diskutil", hint="lsblk is Linux-only. On macOS: bhuppi help disk diskutil", section="Volumes", complete="none")
    c.add("disk/blkid", "block ids", "Show block device attributes by running blkid. Linux.", bin="blkid", unix=["blkid"], shape="bhuppi disk blkid", example="bhuppi disk blkid", platform="linux", fallback="disk diskutil", hint="blkid is Linux-only. On macOS: bhuppi help disk diskutil", section="Volumes", complete="none")
    c.add(
        "disk/free",
        "memory",
        "Show memory. Linux runs free -h. macOS runs vm_stat.",
        bin="free",
        unix=["free", "vm_stat"],
        special="freemem",
        shape="bhuppi disk free",
        example="bhuppi disk free",
        when="You want a quick memory picture.",
        rule="free and vm_stat are different programs. --dry-run prints the one this OS runs.",
        desc="There is no translation of free's columns into vm_stat.",
        not_this="Not a Linux free binary shipped inside bhuppi.",
        runs_linux="free -h",
        runs_darwin="vm_stat",
        also_linux="free -h",
        also_macos="vm_stat",
        section="Look",
        complete="none",
        user_args="none",
    )
    c.add("disk/sync", "flush disks", "Flush filesystem buffers by running sync.", bin="sync", unix=["sync"], shape="bhuppi disk sync", example="bhuppi disk sync", argv=["sync"], user_args="none", special="fixed", section="Volumes", complete="none", not_this="Not a reboot.")
    c.add(
        "disk/fdisk",
        "list partitions",
        "List partitions by running fdisk -l. Linux. Writes are outside this page.",
        bin="fdisk",
        unix=["fdisk"],
        special="fixed",
        argv=["fdisk", "-l"],
        platform="linux",
        fallback="disk diskutil",
        hint="fdisk -l is the Linux list form. On macOS: bhuppi help disk diskutil",
        shape="bhuppi disk fdisk",
        example="bhuppi disk fdisk",
        runs="fdisk -l",
        user_args="none",
        not_this="Not an interactive partition editor. Extra arguments are refused so this page cannot write a table.",
        section="Volumes",
        complete="none",
    )


def auth(c: Cat) -> None:
    rows = [
        ("who", "who is logged in", "who", "who", "Show logged-in users by running who.", "bhuppi auth who", "bhuppi auth who", 0),
        ("w", "who and what", "w", "w", "Show who is logged in and what they run, by running w.", "bhuppi auth w", "bhuppi auth w", 0),
        ("id", "user and groups", "id", "id", "Print user and group ids by running id.", "bhuppi auth id", "bhuppi auth id", 0),
        ("whoami", "current user", "whoami", "whoami", "Print the effective user by running whoami.", "bhuppi auth whoami", "bhuppi auth whoami", 0),
        ("groups", "group list", "groups", "groups", "Print groups by running groups.", "bhuppi auth groups", "bhuppi auth groups", 0),
        ("last", "login history", "last", "last", "Show recent logins by running last.", "bhuppi auth last", "bhuppi auth last", 0),
        ("getent", "name service lookup", "getent", "getent", "Look up a name-service entry by running getent.", "bhuppi auth getent passwd", "bhuppi auth getent passwd", 1),
    ]
    for name, title, bin_name, unix, what, shape, example, min_args in rows:
        c.add(f"auth/{name}", title, what, bin=bin_name, unix=[unix], shape=shape, example=example, min_args=min_args, section="Look", complete="none")
    c.add(
        "auth/sudo",
        "run as another user",
        "Run one command with sudo.",
        bin="sudo",
        unix=["sudo"],
        shape="bhuppi auth sudo CMD...",
        example="bhuppi auth sudo whoami",
        when="You want a command re-run with sudo and you will see the argv first if you dry-run.",
        rule="At least one argument is required. bhuppi does not start a root shell for you.",
        desc="The argv is sudo plus your arguments. --dry-run prints them and does not prompt.",
        not_this="Not sudo -s unless you typed -s. Not a sudoers editor.",
        runs="sudo whoami",
        min_args=1,
        complete="none",
        section="Change",
    )
    c.add("auth/su", "substitute user", "Substitute user by running su with the arguments you type.", bin="su", unix=["su"], shape="bhuppi auth su [USER]", example="bhuppi auth su - root", min_args=1, not_this="Not an automatic root shell. You type the arguments.", section="Change", complete="none")
    c.add("auth/passwd", "change your password", "Change a password by running passwd.", bin="passwd", unix=["passwd"], shape="bhuppi auth passwd [USER]", example="bhuppi auth passwd", not_this="Not a password cracker. Not a hash dump.", section="Change", complete="none", min_args=0)
    c.add("auth/useradd", "add a user", "Add a user by running useradd. Linux.", bin="useradd", unix=["useradd"], shape="bhuppi auth useradd USER", example="bhuppi auth useradd ada", platform="linux", fallback="auth dscl", hint="useradd is Linux-only. On macOS: bhuppi help auth dscl", min_args=1, not_this="Not a bulk account creator.", section="Change", complete="none")
    c.add(
        "auth/dscl",
        "directory service read",
        "Read directory service data with dscl. macOS. The no-argument form lists users.",
        bin="dscl",
        unix=["dscl"],
        shape="bhuppi auth dscl [ARGS]",
        example="bhuppi auth dscl",
        argv=["dscl", ".", "-list", "/Users"],
        user_args="default",
        platform="darwin",
        fallback="auth useradd",
        hint="dscl is macOS. On Linux: bhuppi help auth useradd",
        runs="dscl . -list /Users",
        not_this="Not a create-admin recipe. With arguments, dscl receives exactly what you typed.",
        section="Change",
        complete="none",
    )


def svc(c: Cat) -> None:
    c.add(
        "svc/systemctl",
        "systemd",
        "status, start, stop, enable, or read logs for a unit. Linux.",
        bin="systemctl",
        unix=["systemctl"],
        delivery="language-mode",
        platform="linux",
        fallback="svc launchctl",
        hint="systemctl is Linux-only. On macOS: bhuppi help svc launchctl",
        shape="bhuppi svc systemctl status|start|stop|enable|logs UNIT",
        example="bhuppi svc systemctl status ssh",
        when="This machine runs systemd and the unit is one you administer.",
        rule="The child is the systemctl verb. logs runs journalctl -u, not a search.",
        desc="On macOS the missing binary exits 2 and points at launchctl.",
        not_this="Not a search for malicious units. Not a reboot.",
        section="Services",
    )
    for verb in ("status", "start", "stop", "enable"):
        c.add(
            f"svc/systemctl/{verb}",
            f"{verb} a unit",
            f"{verb.capitalize()} a systemd unit.",
            bin="systemctl",
            shape=f"bhuppi svc systemctl {verb} UNIT",
            example=f"bhuppi svc systemctl {verb} ssh",
            argv=["systemctl", verb],
            runs=f"systemctl {verb} ssh",
            platform="linux",
            fallback="svc launchctl",
            min_args=1,
            complete="none",
        )
    c.add(
        "svc/systemctl/logs",
        "unit logs",
        "Show a unit's journal with journalctl -u.",
        bin="journalctl",
        shape="bhuppi svc systemctl logs UNIT",
        example="bhuppi svc systemctl logs ssh",
        argv=["journalctl", "--no-pager", "-u"],
        runs="journalctl --no-pager -u ssh",
        platform="linux",
        fallback="logs unified",
        min_args=1,
        complete="none",
        not_this="Not a brute-force hunt. Not a search across every unit.",
    )
    c.add(
        "svc/launchctl",
        "launchd",
        "List launchd jobs, or load a unit you own into your gui domain. macOS.",
        bin="launchctl",
        unix=["launchctl"],
        delivery="capability",
        platform="darwin",
        fallback="svc systemctl",
        hint="launchctl is macOS. On Linux: bhuppi help svc systemctl",
        shape="bhuppi svc launchctl list|load",
        example="bhuppi svc launchctl list",
        when="You want launchd on this Mac, for agents you own.",
        rule="load uses launchctl bootstrap gui/UID. It does not target the system domain.",
        desc="list is launchctl list. load takes a plist path.",
        not_this="Not bootout of system services. Not a reboot.",
        section="Services",
    )
    c.add("svc/launchctl/list", "list jobs", "List launchd jobs.", bin="launchctl", special="fixed", argv=["launchctl", "list"], shape="bhuppi svc launchctl list", example="bhuppi svc launchctl list", runs="launchctl list", platform="darwin", fallback="svc systemctl", user_args="none", complete="none")
    c.add(
        "svc/launchctl/load",
        "load your unit",
        "Bootstrap a plist into gui/UID, the domain for units you own.",
        bin="launchctl",
        special="launchctl-load",
        shape="bhuppi svc launchctl load PLIST",
        example="bhuppi svc launchctl load ~/Library/LaunchAgents/mine.plist",
        runs="launchctl bootstrap gui/UID ~/Library/LaunchAgents/mine.plist",
        platform="darwin",
        fallback="svc systemctl",
        min_args=1,
        complete="files",
        user_args="none",
        not_this="Not sudo launchctl in the system domain.",
    )
    c.add("svc/cron", "your crontab", "Show your crontab. crontab -l when you pass no flags.", bin="crontab", unix=["crontab"], shape="bhuppi svc cron", example="bhuppi svc cron", argv=["crontab", "-l"], user_args="default", runs="crontab -l", desc="No arguments runs crontab -l. If you pass arguments, they replace that default: crontab -e only when you type -e.", not_this="Not a system-wide cron installer. Not a reboot.", section="Services", complete="none")
    c.add("svc/service", "service helper", "Run the service helper, where that binary exists (often Linux).", bin="service", unix=["service"], shape="bhuppi svc service NAME ACTION", example="bhuppi svc service ssh status", platform="linux", fallback="svc launchctl", hint="service is a Linux init helper. On macOS: bhuppi help svc launchctl", min_args=1, section="Services", complete="none")
    c.add("svc/at", "one-shot at", "Schedule a one-shot job by running at, when at is installed.", bin="at", unix=["at"], shape="bhuppi svc at TIME", example="bhuppi svc at now", optional=True, hint="at is often absent. Install the at package from your OS if you need it.", not_this="Not a persistent backdoor. You type the time.", section="Services", complete="none", min_args=1)


def logs(c: Cat) -> None:
    c.add(
        "logs/journal",
        "systemd journal",
        "Show the last 50 journal lines. Linux.",
        bin="journalctl",
        unix=["journalctl"],
        shape="bhuppi logs journal [ARGS]",
        example="bhuppi logs journal",
        argv=["journalctl", "-n", "50", "--no-pager"],
        user_args="default",
        platform="linux",
        fallback="logs unified",
        hint="journalctl is Linux-only. On macOS: bhuppi help logs unified",
        runs="journalctl -n 50 --no-pager",
        desc="No arguments prints a short page. Arguments you type replace that default and go to journalctl as you wrote them.",
        not_this="Not a brute-force hunt. Not a search for failed passwords unless you typed that filter yourself.",
        section="Read",
        complete="none",
    )
    c.add("logs/kernel", "kernel ring", "Print the kernel ring buffer by running dmesg.", bin="dmesg", unix=["dmesg"], shape="bhuppi logs kernel", example="bhuppi logs kernel", argv=["dmesg"], section="Read", complete="none", not_this="Not a crash exploit. A reader only.")
    c.add("logs/logger", "write a log line", "Write a syslog line by running logger.", bin="logger", unix=["logger"], shape="bhuppi logs logger MESSAGE", example="bhuppi logs logger hello", min_args=1, section="Write", complete="none")
    c.add(
        "logs/unified",
        "unified log",
        "Show recent unified log entries with log show. macOS.",
        bin="log",
        unix=["log"],
        shape="bhuppi logs unified",
        example="bhuppi logs unified",
        argv=["log", "show", "--last", "5m"],
        special="fixed",
        platform="darwin",
        fallback="logs journal",
        hint="log show is macOS. On Linux: bhuppi help logs journal",
        runs="log show --last 5m",
        user_args="none",
        not_this="Not a predicate you did not type. This page is a five-minute window only.",
        section="Read",
        complete="none",
    )


def host(c: Cat) -> None:
    c.add("host/os", "operating system", "Print uname -a.", bin="uname", unix=["uname", "sw_vers"], special="os", shape="bhuppi host os", example="bhuppi host os", runs="uname -a", also_macos="sw_vers", desc="The argv is uname -a. sw_vers is the macOS peer named on this page; it is not silently appended. explain sw_vers points here.", section="Machine", complete="none", user_args="none")
    c.add("host/env", "environment", "Print the environment by running env.", bin="env", unix=["env", "printenv"], shape="bhuppi host env", example="bhuppi host env", argv=["env"], section="Machine", complete="none", not_this="Not a secret store. The output is your environment; do not pipe it into a ticket.")
    c.add(
        "host/pkg",
        "packages",
        "Search or install with apt, dnf, yum, or brew — whichever this machine has.",
        delivery="capability",
        unix=["apt", "apt-get", "yum", "dnf", "brew"],
        shape="bhuppi host pkg search|install NAME",
        example="bhuppi host pkg search ripgrep",
        when="You want the native package manager, named in the dry-run.",
        rule="brew is not prefixed with sudo. apt, dnf, and yum install are prefixed with sudo because that is how those tools install. --dry-run shows the chosen manager.",
        desc="search and install are the children. The manager is detected: brew on macOS, then apt-get, dnf, or yum.",
        not_this="Not a new package manager. Not Homebrew rewritten into apt.",
        section="Packages",
    )
    c.add("host/pkg/search", "search packages", "Search the native package index.", special="pkg-search", shape="bhuppi host pkg search NAME", example="bhuppi host pkg search ripgrep", runs="brew search ripgrep", min_args=1, complete="none", user_args="none", also_linux="apt-cache search ripgrep", also_macos="brew search ripgrep")
    c.add("host/pkg/install", "install a package", "Install one named package with the native manager.", special="pkg-install", shape="bhuppi host pkg install NAME", example="bhuppi host pkg install ripgrep", runs="brew install ripgrep", min_args=1, complete="none", user_args="none", also_linux="sudo apt-get install ripgrep", also_macos="brew install ripgrep", not_this="Not a silent sudo in front of brew.")
    for name, title, bin_name, unix, what, example, argv in [
        ("date", "date and time", "date", "date", "Print the date by running date.", "bhuppi host date", ["date"]),
        ("uptime", "uptime", "uptime", "uptime", "Print uptime by running uptime.", "bhuppi host uptime", ["uptime"]),
        ("hostname", "hostname", "hostname", "hostname", "Print the hostname by running hostname.", "bhuppi host hostname", ["hostname"]),
        ("sysctl", "kernel parameters", "sysctl", "sysctl", "Read kernel parameters by running sysctl.", "bhuppi host sysctl -a", ["sysctl"]),
        ("lscpu", "CPU info", "lscpu", "lscpu", "Print CPU info by running lscpu, when it exists.", "bhuppi host lscpu", ["lscpu"]),
        ("arch", "architecture", "arch", "arch", "Print the architecture by running arch or uname -m.", "bhuppi host arch", ["arch"]),
        ("hostnamectl", "hostname status", "hostnamectl", "hostnamectl", "Show hostnamectl status. Linux.", "bhuppi host hostnamectl", ["hostnamectl"]),
    ]:
        kw = dict(bin=bin_name, unix=[unix], shape=f"bhuppi host {name}", example=example, argv=argv, section="Machine", complete="none", min_args=0)
        if name == "date":
            kw["also_linux"] = "date -d '2020-01-01' is GNU"
            kw["also_macos"] = "date -j -f is BSD; bhuppi does not translate -d"
            kw["desc"] = "The argv is date plus flags you type. GNU -d and BSD -j -f are different and stay different."
        if name == "lscpu":
            kw["platform"] = "linux"
            kw["fallback"] = "host sysctl"
            kw["hint"] = "lscpu is usually Linux. On macOS: bhuppi host sysctl -n machdep.cpu.brand_string is sysctl, which you type."
            kw["optional"] = True
        if name == "hostnamectl":
            kw["platform"] = "linux"
            kw["fallback"] = "host hostname"
            kw["hint"] = "hostnamectl is Linux systemd. On macOS: bhuppi help host hostname"
        if name == "arch":
            kw["special"] = "arch"
            kw.pop("argv")
            kw["user_args"] = "none"
            kw["runs_linux"] = "arch"
            kw["runs_darwin"] = "uname -m"
        c.add(f"host/{name}", title, what, **kw)


def data(c: Cat) -> None:
    c.add("data/jq", "jq", "Query JSON with jq. Four small shapes, then the real jq.", bin="jq", unix=["jq"], delivery="language-mode", shape="bhuppi data jq get|filter|keys|slurp …", example="bhuppi data jq get .version app.json", when="You have JSON and a filter.", rule="Each child execs jq. A missing jq exits 2 with an install hint. There is no fake jq.", desc="get, filter, keys, and slurp. Bare `bhuppi data jq` prints this page.", not_this="Not a JSON tutorial. Not a second implementation.", hint="brew install jq    or    apt install jq", section="JSON")
    c.add("data/jq/get", "read one path", "Print one jq path, raw.", bin="jq", shape="bhuppi data jq get PATH FILE", example="bhuppi data jq get .version app.json", argv=["jq", "-r"], runs="jq -r .version app.json", complete="files", complete_skip=1, min_args=2)
    c.add("data/jq/filter", "run a filter", "Run a jq filter you write.", bin="jq", shape="bhuppi data jq filter FILTER FILE", example="bhuppi data jq filter '.items[]' app.json", argv=["jq"], runs="jq '.items[]' app.json", complete="files", complete_skip=1, min_args=2)
    c.add("data/jq/keys", "list keys", "List keys of a JSON object.", bin="jq", shape="bhuppi data jq keys FILE", example="bhuppi data jq keys app.json", argv=["jq", "keys"], runs="jq keys app.json", complete="files", min_args=1)
    c.add("data/jq/slurp", "slurp inputs", "Read inputs as one array with jq -s.", bin="jq", shape="bhuppi data jq slurp FILTER FILES", example="bhuppi data jq slurp . a.json", argv=["jq", "-s"], runs="jq -s . a.json", complete="files", complete_skip=1, min_args=1)
    c.add("data/yq", "yq", "Query YAML with yq, same four shapes as jq.", bin="yq", unix=["yq"], delivery="language-mode", shape="bhuppi data yq get|filter|keys|slurp …", example="bhuppi data yq get .version app.yaml", hint="brew install yq    or    the yq your distro packages. mikefarah/yq and python yq differ; bhuppi execs whichever yq is first on PATH and does not translate filters.", not_this="Not a translator between yq dialects.", section="JSON", optional=True)
    for name, argv, shape, example, runs, min_args, skip in [
        ("get", ["yq", "-r"], "bhuppi data yq get PATH FILE", "bhuppi data yq get .version app.yaml", "yq -r .version app.yaml", 2, 1),
        ("filter", ["yq"], "bhuppi data yq filter FILTER FILE", "bhuppi data yq filter '.a' app.yaml", "yq '.a' app.yaml", 2, 1),
        ("keys", ["yq", "keys"], "bhuppi data yq keys FILE", "bhuppi data yq keys app.yaml", "yq keys app.yaml", 1, 0),
        ("slurp", ["yq", "-s"], "bhuppi data yq slurp FILTER FILES", "bhuppi data yq slurp . a.yaml", "yq -s . a.yaml", 1, 1),
    ]:
        c.add(f"data/yq/{name}", name, f"yq {name}, same shape as jq {name}.", bin="yq", shape=shape, example=example, argv=argv, runs=runs, min_args=min_args, complete="files", complete_skip=skip)
    c.add(
        "data/openssl",
        "openssl",
        "Key, CSR, certificate text, verify, or a handshake to a host you operate.",
        bin="openssl",
        unix=["openssl"],
        delivery="language-mode",
        shape="bhuppi data openssl key|csr|cert|verify|handshake …",
        example="bhuppi data openssl handshake example.com",
        when="You administer the key or the host.",
        rule="Five children. No s_server, no intercepting proxy, no extra openssl subcommand.",
        desc="key writes a new RSA key. csr builds a request. cert prints x509 text. verify checks a chain. handshake is s_client and then stdin is closed so it can exit.",
        not_this="Not an interception kit. Not a private-key exfiltration recipe.",
        section="TLS",
    )
    c.add("data/openssl/key", "write a key", "Generate an RSA key with openssl genpkey.", bin="openssl", special="openssl-key", shape="bhuppi data openssl key FILE", example="bhuppi data openssl key host.pem", runs="openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out host.pem", min_args=1, complete="files", user_args="none", not_this="Not a copy of someone else's key.")
    c.add("data/openssl/csr", "write a CSR", "Create a certificate signing request.", bin="openssl", special="openssl-csr", shape="bhuppi data openssl csr KEY SUBJ OUT", example="bhuppi data openssl csr host.pem /CN=example.com host.csr", runs="openssl req -new -key host.pem -subj /CN=example.com -out host.csr", min_args=3, complete="files", complete_skip=2, user_args="none")
    c.add("data/openssl/cert", "show a certificate", "Print certificate text.", bin="openssl", special="openssl-cert", shape="bhuppi data openssl cert FILE", example="bhuppi data openssl cert host.crt", runs="openssl x509 -in host.crt -noout -text", min_args=1, complete="files", user_args="none")
    c.add("data/openssl/verify", "verify a chain", "Verify a certificate against a CA file.", bin="openssl", special="openssl-verify", shape="bhuppi data openssl verify CA CERT", example="bhuppi data openssl verify ca.pem host.crt", runs="openssl verify -CAfile ca.pem host.crt", min_args=2, complete="files", user_args="none")
    c.add("data/openssl/handshake", "TLS handshake", "Handshake with a host you operate, then stop.", bin="openssl", special="openssl-handshake", stdin_null=True, shape="bhuppi data openssl handshake HOST", example="bhuppi data openssl handshake example.com", runs="openssl s_client -connect example.com:443 -servername example.com", min_args=1, complete="none", user_args="none", not_this="Not an intercepting proxy. Not s_server.")
    c.add(
        "data/hash",
        "checksum a file",
        "Hash a file. Default SHA-256. md5 and sha1 are peers you can name.",
        bin="sha256sum",
        unix=["md5", "md5sum", "sha1sum", "sha256sum", "sha512sum", "shasum"],
        special="hash",
        delivery="peer",
        shape="bhuppi data hash [md5|sha1|sha256|sha512] FILE",
        example="bhuppi data hash README",
        when="You want a checksum of a file you have.",
        rule="Linux uses the coreutils *sum binaries. macOS uses shasum -a and md5. --dry-run prints the peer this OS runs. Names are not rewritten in the help into a single fictional tool.",
        desc="Default is SHA-256. md5sum and md5 are the MD5 peer pair.",
        not_this="Not a password hash cracker. Not hashcat.",
        runs_linux="sha256sum README",
        runs_darwin="shasum -a 256 README",
        also_linux="md5sum README",
        also_macos="md5 README",
        section="Hash",
        complete="files",
        complete_skip=0,
        min_args=1,
    )
    for name, title, bin_name, unix, what, example in [
        ("base64", "base64", "base64", "base64", "Encode or decode with base64. Pass -d yourself to decode.", "bhuppi data base64 FILE"),
        ("xxd", "hex dump", "xxd", "xxd", "Hex-dump a file with xxd.", "bhuppi data xxd FILE"),
        ("hexdump", "hex dump", "hexdump", "hexdump", "Hex-dump a file with hexdump.", "bhuppi data hexdump FILE"),
        ("od", "octal dump", "od", "od", "Dump a file with od.", "bhuppi data od FILE"),
        ("cksum", "CRC checksum", "cksum", "cksum", "CRC checksum by running cksum. This is not SHA-256; that is data hash.", "bhuppi data cksum FILE"),
    ]:
        c.add(f"data/{name}", title, what, bin=bin_name, unix=[unix], shape=f"bhuppi data {name} FILE", example=example, section="Bytes", complete="files", min_args=1)


def dev(c: Cat) -> None:
    rows = [
        ("git", "git", "git", "git", "Run git with the arguments you type.", "bhuppi dev git status", 1),
        ("gcc", "gcc", "gcc", "gcc", "Compile with gcc.", "bhuppi dev gcc -o hi hi.c", 1),
        ("gxx", "g++", "g++", "g++", "Compile C++ with g++.", "bhuppi dev gxx -o hi hi.cpp", 1),
        ("clang", "clang", "clang", "clang", "Compile with clang.", "bhuppi dev clang -o hi hi.c", 1),
        ("make", "make", "make", "make", "Run make. explain make is this tool, not files make.", "bhuppi dev make", 0),
        ("docker", "docker", "docker", "docker", "Run docker with the arguments you type. No mounts or capabilities are added.", "bhuppi dev docker ps", 1),
        ("ldd", "shared libraries", "ldd", "ldd", "List shared libraries with ldd. Linux.", "bhuppi dev ldd FILE", 1),
        ("otool", "Mach-O libraries", "otool", "otool", "Show Mach-O load commands with otool -L when you pass no further flags.", "bhuppi dev otool FILE", 1),
        ("nm", "symbol names", "nm", "nm", "List symbols with nm.", "bhuppi dev nm FILE", 1),
        ("objdump", "object dump", "objdump", "objdump", "Dump an object file with objdump.", "bhuppi dev objdump -d FILE", 1),
        ("strip", "strip symbols", "strip", "strip", "Strip symbols with strip.", "bhuppi dev strip FILE", 1),
        ("patch", "apply a diff", "patch", "patch", "Apply a diff with patch.", "bhuppi dev patch -p1", 1),
        ("node", "node", "node", "node", "Run node.", "bhuppi dev node script.js", 1),
        ("go", "go tool", "go", "go", "Run the go tool. explain go is this compiler toolchain, not files go.", "bhuppi dev go test ./...", 1),
        ("cargo", "cargo", "cargo", "cargo", "Run cargo.", "bhuppi dev cargo build", 1),
    ]
    for name, title, bin_name, unix, what, example, min_args in rows:
        kw = dict(bin=bin_name, unix=[unix], shape=f"bhuppi dev {name} [ARGS]", example=example, min_args=min_args, section="Build", complete="both" if name in ("gcc", "clang", "gxx", "strip", "nm", "objdump", "ldd", "otool") else "none")
        if name == "ldd":
            kw["platform"] = "linux"
            kw["fallback"] = "dev otool"
            kw["hint"] = "ldd is Linux/ELF. On macOS: bhuppi help dev otool"
        if name == "otool":
            kw["platform"] = "darwin"
            kw["fallback"] = "dev ldd"
            kw["hint"] = "otool is macOS. On Linux: bhuppi help dev ldd"
            kw["argv"] = ["otool", "-L"]
            kw["user_args"] = "default"
            kw["desc"] = "No arguments is not valid. One file and no flags runs otool -L FILE. If you pass flags, they replace that default."
            kw["special"] = "otool"
        if name == "make":
            kw["explain_line"] = "dev make — the make tool. files make is mkdir or touch."
        if name == "go":
            kw["explain_line"] = "dev go — the Go toolchain. files go and folders go are cd."
        if name == "gxx":
            kw["shape"] = "bhuppi dev gxx [ARGS]"
        c.add(f"dev/{name}", title, what, **kw)


def docs(c: Cat) -> None:
    c.add("docs/man", "manual page", "Open a manual with man.", bin="man", unix=["man"], shape="bhuppi docs man PAGE", example="bhuppi docs man ls", min_args=1, complete="none", section="Manuals")
    c.add("docs/which", "which binary", "Print the PATH entry for a command by running which.", bin="which", unix=["which"], shape="bhuppi docs which NAME", example="bhuppi docs which ls", min_args=1, complete="none", section="Manuals", not_this="Not the shell builtin type. type is a skip; see lang bash builtin.")
    c.add("docs/whereis", "whereis", "Locate a binary, source, and man page by running whereis.", bin="whereis", unix=["whereis"], shape="bhuppi docs whereis NAME", example="bhuppi docs whereis ls", min_args=1, complete="none", section="Manuals")
    c.add("docs/apropos", "search manuals", "Search man page names by running apropos.", bin="apropos", unix=["apropos"], shape="bhuppi docs apropos TEXT", example="bhuppi docs apropos copy", min_args=1, complete="none", section="Manuals")
    c.add("docs/whatis", "one-line manuals", "Show the one-line man description by running whatis.", bin="whatis", unix=["whatis"], shape="bhuppi docs whatis PAGE", example="bhuppi docs whatis ls", min_args=1, complete="none", section="Manuals")
    c.add("docs/info", "info manual", "Open an info manual by running info.", bin="info", unix=["info"], shape="bhuppi docs info PAGE", example="bhuppi docs info coreutils", min_args=1, complete="none", section="Manuals", optional=True, hint="info is often part of texinfo.")
    c.add(
        "docs/security",
        "macOS security tool is docs-only",
        "security(1) is the macOS keychain and certificate tool. bhuppi names it and does not run it.",
        bin="security",
        unix=["security"],
        delivery="docs-only",
        shape="bhuppi docs security",
        example="bhuppi explain security",
        when="You want the official manual for the keychain tool.",
        rule="No flags are accepted and security is not executed.",
        desc="Keychain items, passwords, and certificates stay in Apple's tool. This page is a pointer.",
        not_this="Not a keychain dump. Not a certificate export recipe.",
        docs="https://ss64.com/osx/security.html",
        platform="darwin",
        complete="none",
        user_args="none",
    )


def meta(c: Cat) -> None:
    pages = [
        ("help", "progressive help", "Static help at every level. Same renderer as --help. No model.", "bhuppi help [GROUP [CMD [SUB]]]", "bhuppi help files", "You want the shape before you run anything.", "help and --help call one renderer. h is the only alias."),
        ("recipe", "safe pipelines", "Print a named, copy-safe pipeline. Add --run to execute it.", "bhuppi recipe [NAME] [--run]", "bhuppi recipe disk-hogs", "You want a known-safe composition, visible before it runs.", "Default is print only. --run executes. --dry-run prints the same pipeline."),
        ("doctor", "what this machine has", "Show OS, flavor probes, and whether each cataloged binary is on PATH.", "bhuppi doctor", "bhuppi doctor", "A tool is missing and you want the honest table.", "Present, missing, GNU, BSD, Linux-only, and darwin-only stay labeled. Nothing is installed by doctor."),
        ("explain", "Unix name to job", "Map a Unix name onto the job tree. This is the A–Z hatch.", "bhuppi explain NAME", "bhuppi explain awk", "You remember awk and want the job path.", "The first line is the job path, then one line of what it is. Skips and docs-only names say so. When the book has the command, a guide line points at bhuppi atlas."),
        ("atlas", "command atlas", "Long guide for a command from the book: syntax, anatomy, origins, and the book's use cases.", "bhuppi atlas [NAME [SECTION]]", "bhuppi atlas awk", "You want the long guide, not the one-line job pointer.", "With no name, bhuppi atlas is the index. A name opens that command. A further word jumps to a heading such as fields. Type a use-case number, then Enter, to copy its command · Enter alone moves one line · Space next page · q stop. Missing names list the closest. This is not a 16th job group."),
        ("color", "color switch", "Turn sober color on or off for later commands, or print the current setting.", "bhuppi color [on|off]", "bhuppi color off", "You want plain text in new shells, or you want color back.", "Default is on. The choice is stored in ~/.config/bhuppi/color. --plain forces plain for one invocation. NO_COLOR, when set and non-empty, forces plain."),
        ("completion", "tab completion", "Print completion code for zsh, bash, or powershell.", "bhuppi completion zsh|bash|powershell", "bhuppi completion zsh", "You want Tab to split commands, files, and folders.", "Three groups. Folders end with /. files show omits folders. folders go omits files. There is no single paths bucket."),
        ("shell-init", "shell hook", "Print the hook and completion for zsh, bash, or powershell.", "bhuppi shell-init zsh|bash|powershell", "bhuppi shell-init zsh", "You want files go to cd, and you want completion.", "The installer writes a source line. You do not need to eval it yourself. The hook cds; it does not export or alias."),
        ("version", "version", "Print the bhuppi version.", "bhuppi version", "bhuppi version", "You want the build's version string.", "version does not exec a Unix tool."),
        (
            "config",
            "local settings",
            "Store settings on this machine. Reputation keys are the first setting.",
            "bhuppi config reputation status",
            "bhuppi config reputation status",
            "You want a later command to use a key without putting that key on the command line.",
            "Three commands: bhuppi config reputation set vt, bhuppi config reputation set abuseip, and bhuppi config reputation status. set reads the key from the terminal with echo off, or from stdin when stdin is not a terminal. The key is not a command argument. status prints set or missing and never prints the key. The file is ~/.config/bhuppi/reputation.json.",
        ),
        (
            "ai-agent",
            "local agent review",
            "One structured page for a local AI agent on this machine: processes, connections, sensitive paths, persistence, and scheduled tasks.",
            "bhuppi ai-agent review [NAME|PID]",
            "bhuppi ai-agent review cursor",
            "You want a 360 review of Cursor, Claude, Codex, or another local agent without opening its files.",
            "Paths only. File contents, keychain items, certificate material, and environment variables are not printed. The review is this machine only.",
        ),
    ]
    for name, title, what, shape, example, when, rule in pages:
        c.meta.append(dict(
            name=name,
            title=title,
            what=what,
            when=when,
            rule=rule,
            desc=what + " " + rule,
            not_this="Not a job group. Not a 16th catalog section.",
            more=f"bhuppi help {name} · bhuppi {example.split()[1] if name != 'help' else 'help files'}",
            shape=shape,
            example=example,
            delivery="meta",
            complete="none",
        ))
    for page in c.meta:
        if page["name"] != "ai-agent":
            continue
        page["not_this"] = "Not a secret dump. Not a remote scan. Not dtruss. Not a 16th job group."
        page["more"] = "bhuppi help ai-agent review · bhuppi ai-agent review cursor"
        page["children"] = [dict(
            name="review",
            title="360 review",
            what="Review one local AI agent, or every matching agent, on a single page.",
            when="You want the process tree, who it talks to, which sensitive paths it has open, and whether it starts itself.",
            rule="A name is one of cursor, claude, codex, antigravity, deepseek, qwen, chatgpt, copilot, windsurf, ollama. No name reviews each match. Every match is on the page. A pid reviews that process and its children. --dry-run prints ps, lsof, crontab, and launchctl or systemctl, and does not run them.",
            desc="Sections, in order: processes, connections, sensitive paths, persistence, scheduled tasks. A red ! marks an abnormal line. Secrets are not printed.",
            not_this="Not a secret dump. Not a remote scan. Not dtruss.",
            more="bhuppi ai-agent review cursor · bhuppi help ai-agent review",
            shape="bhuppi ai-agent review [NAME|PID]",
            example="bhuppi ai-agent review cursor",
            delivery="meta",
            complete="none",
        )]
    for page in c.meta:
        if page["name"] != "config":
            continue
        page["not_this"] = "Not a job group. Not a 16th catalog section. Reputation-check is not the default. Keys are not command arguments. Private addresses are not sent to either service."
        page["more"] = "bhuppi help config · bhuppi config reputation status"
        page["children"] = [dict(
            name="reputation",
            title="service keys",
            what="Save a VirusTotal or AbuseIPDB key, or see whether one is stored.",
            when="You want bhuppi net connections reputation-check to use a key, or you want to see whether a key is already stored.",
            rule="bhuppi config reputation set vt reads the VirusTotal key with echo off. bhuppi config reputation set abuseip reads the AbuseIPDB key the same way. bhuppi config reputation status prints set or missing. The key is not a command argument. Neither command prints the key.",
            desc="The file is ~/.config/bhuppi/reputation.json, mode 0600. Loopback and private addresses are not sent to either service.",
            not_this="Reputation-check is not the default. Keys are not command arguments. Private addresses are not sent to either service.",
            more="bhuppi config reputation set vt · bhuppi config reputation set abuseip · bhuppi config reputation status",
            shape="bhuppi config reputation status",
            example="bhuppi config reputation status",
            examples=[
                "bhuppi config reputation set vt",
                "bhuppi config reputation set abuseip",
            ],
            delivery="meta",
            complete="none",
            section="Keys",
        )]


def recipes(c: Cat) -> None:
    def R(name, title, what, shape, example, desc, not_this, steps, **kw):
        row = dict(
            name=name,
            title=title,
            what=what,
            when=kw.get("when", "You want this exact pipeline and you want to read it first."),
            rule="The pipeline is fixed. Arguments fill placeholders only. Default prints the line; --run executes it.",
            desc=desc,
            not_this=not_this,
            more=f"bhuppi help recipe {name}",
            shape=shape,
            example=example,
            delivery="recipe",
            unix=[name],
            steps=steps,
        )
        for key in ("steps_linux", "steps_darwin", "steps_windows", "also_linux", "also_macos", "runs", "runs_linux", "runs_darwin"):
            if key in kw:
                row[key] = kw[key]
        c.recipes.append(row)

    R(
        "disk-hogs",
        "largest directories",
        "Show which directories directly under DIR use the most space.",
        "bhuppi recipe disk-hogs [DIR]",
        "bhuppi recipe disk-hogs .",
        "Prints du, sort, and head. DIR defaults to `.`. --run executes the pipeline.",
        "Not a delete of large files.",
        steps=[["du", "-h", "-d", "1", "{dir}"], ["sort", "-hr"], ["head", "-n", "20"]],
        steps_linux=[["du", "-h", "--max-depth=1", "{dir}"], ["sort", "-hr"], ["head", "-n", "20"]],
        also_linux="du -h --max-depth=1 . | sort -hr | head -n 20",
        also_macos="du -h -d 1 . | sort -hr | head -n 20",
        runs_linux="du -h --max-depth=1 . | sort -hr | head -n 20",
        runs_darwin="du -h -d 1 . | sort -hr | head -n 20",
    )
    R(
        "who-listens",
        "listeners on this machine",
        "Show TCP listeners on this machine.",
        "bhuppi recipe who-listens",
        "bhuppi recipe who-listens",
        "Linux prints ss -lptn. macOS prints lsof for LISTEN sockets.",
        "Not a scan of other hosts.",
        steps=[["ss", "-lptn"]],
        steps_darwin=[["lsof", "-nP", "-iTCP", "-sTCP:LISTEN"]],
        steps_linux=[["ss", "-lptn"]],
        also_linux="ss -lptn",
        also_macos="lsof -nP -iTCP -sTCP:LISTEN",
    )
    R(
        "newest-in",
        "newest files",
        "List the newest files under DIR.",
        "bhuppi recipe newest-in [DIR]",
        "bhuppi recipe newest-in .",
        "GNU find may use -printf. BSD find uses stat. --dry-run prints the form for this OS.",
        "Not a wipe of old files.",
        steps=[["find", "{dir}", "-type", "f", "-exec", "ls", "-lt", "{}", "+"]],
        steps_linux=[["find", "{dir}", "-type", "f", "-printf", "%T@ %p\\n"], ["sort", "-nr"], ["head", "-n", "20"]],
        steps_darwin=[["find", "{dir}", "-type", "f", "-exec", "stat", "-f", "%m %N", "{}", "+"], ["sort", "-nr"], ["head", "-n", "20"]],
        also_linux="find . -type f -printf '%T@ %p\\n' | sort -nr | head -n 20",
        also_macos="find . -type f -exec stat -f '%m %N' {} + | sort -nr | head -n 20",
    )
    R(
        "checksum",
        "SHA-256 a file",
        "Print the SHA-256 checksum of FILE.",
        "bhuppi recipe checksum FILE",
        "bhuppi recipe checksum README",
        "Linux runs sha256sum. macOS runs shasum -a 256.",
        "Not MD5 unless you use data hash md5. Not a cracker.",
        steps=[["sha256sum", "{file}"]],
        steps_darwin=[["shasum", "-a", "256", "{file}"]],
        steps_linux=[["sha256sum", "{file}"]],
        also_linux="sha256sum FILE",
        also_macos="shasum -a 256 FILE",
    )
    R(
        "tar-dir",
        "tar a directory",
        "Create DIR.tar.gz beside the job by running tar -czf.",
        "bhuppi recipe tar-dir DIR",
        "bhuppi recipe tar-dir logs",
        "The archive name is the directory's base name plus .tar.gz. --run creates it in the current directory.",
        "Not an extract. Not a tar of a path you did not name.",
        steps=[["tar", "-czf", "{archive}", "{dir}"]],
    )
    R(
        "cron-mine",
        "your crontab",
        "Print your user crontab.",
        "bhuppi recipe cron-mine",
        "bhuppi recipe cron-mine",
        "The pipeline is crontab -l. It does not edit.",
        "Not crontab -e unless you run svc cron -e yourself. Not a system cron install.",
        steps=[["crontab", "-l"]],
    )
    R(
        "conn-mine",
        "your TCP connections",
        "Show established TCP connections on this machine.",
        "bhuppi recipe conn-mine",
        "bhuppi recipe conn-mine",
        "Linux uses ss -tp. macOS uses lsof for ESTABLISHED sockets.",
        "Not a scan. Not a packet capture.",
        steps=[["ss", "-tp"]],
        steps_linux=[["ss", "-tp"]],
        steps_darwin=[["lsof", "-nP", "-iTCP", "-sTCP:ESTABLISHED"]],
        also_linux="ss -tp",
        also_macos="lsof -nP -iTCP -sTCP:ESTABLISHED",
    )


def explain_extras(c: Cat) -> None:
    skips = [
        ("export", "Shell builtin. Sets a variable in the current shell. bhuppi does not inject it."),
        ("alias", "Shell builtin. bhuppi does not write aliases into your shell."),
        ("history", "Shell builtin. bhuppi does not read or rewrite history."),
        ("type", "Shell builtin. For a PATH lookup use docs which."),
        ("test", "Shell builtin, also `[`. bhuppi does not wrap it."),
        ("[", "Shell builtin test. Same skip as test."),
        ("source", "Shell builtin. bhuppi does not source files into your shell."),
        ("eval", "Shell builtin. bhuppi does not eval strings for you."),
        ("jobs", "Shell job control. Not injected."),
        ("bg", "Shell job control. Not injected."),
        ("fg", "Shell job control. Not injected."),
        ("clear", "Terminal chrome. Not a job."),
        ("reset", "Terminal chrome. Not a job."),
        ("stty", "Terminal chrome. Not a job."),
        ("tput", "Terminal chrome. Not a job."),
        ("screen", "Terminal multiplexer chrome. Not a job group."),
        ("tmux", "Terminal multiplexer chrome. Not a job group."),
        ("delete", "Not a bhuppi verb. files remove and folders remove are the jobs."),
        ("reboot", "Not a bhuppi job. Surprise reboot recipes are refused."),
        ("shutdown", "Not a bhuppi job. Surprise power recipes are refused."),
        ("halt", "Not a bhuppi job."),
        ("poweroff", "Not a bhuppi job."),
        ("hashcat", "Refused. bhuppi does not wrap password cracking."),
        ("john", "Refused. bhuppi does not wrap password cracking."),
        ("hydra", "Refused. bhuppi does not wrap online password guessing."),
        ("metasploit", "Refused. bhuppi does not wrap exploitation frameworks."),
        ("sqlmap", "Refused. bhuppi does not wrap SQL injection tools."),
    ]
    c.extra(
        "dtruss",
        "proc strace",
        "dtruss traces system calls on macOS with DTrace. SIP blocks it for ordinary use. Docs only: no flags, no exec.",
        "skip",
        "dtruss traces system calls on macOS with DTrace. System Integrity Protection blocks it for ordinary use.",
        when="You typed dtruss on a Mac and want to know why bhuppi will not run it.",
        rule="Skip means help only. The name and purpose are recorded. No flag recipe is printed and nothing is executed.",
        not_this="Not a wrapper. Not a stealth tracer.",
        docs="https://developer.apple.com/library/archive/documentation/Darwin/Conceptual/KernelProgramming/build/build.html",
        example="",
    )
    for name, line in skips:
        c.extra(
            name,
            "lang bash builtin" if name in {"export", "alias", "history", "type", "test", "[", "source", "eval", "jobs", "bg", "fg"} else "",
            line,
            "skip",
            line,
            when="You typed a Unix or shell name that this atlas will not exec.",
            rule="Skip means help only. Nothing is executed and no flag recipe is printed.",
            not_this="Not a wrapper.",
            example="bhuppi lang bash builtin" if name in {"export", "alias", "history", "type", "test"} else "",
        )


def validate(doc: dict) -> None:
    names = [g["name"] for g in doc["groups"]]
    if names != [
        "files", "folders", "text", "lang", "proc", "net", "pack", "disk",
        "auth", "svc", "logs", "host", "data", "dev", "docs",
    ]:
        raise SystemExit(f"group set mismatch: {names}")
    banned = {"delete", "inspect"}
    explain: dict[str, str] = {}

    def walk(node: dict, path: str) -> None:
        for key in ("what", "when", "rule", "desc", "not_this", "shape", "example", "more"):
            if not str(node.get(key, "")).strip():
                raise SystemExit(f"{path or node.get('name')} missing {key}")
        if node.get("name") in banned and path.split(" ")[0] in {"files", "folders", ""}:
            raise SystemExit(f"banned verb {node['name']} at {path}")
        for u in node.get("unix") or []:
            if u in explain and explain[u] != path:
                raise SystemExit(f"duplicate unix {u}: {explain[u]} vs {path}")
            explain[u] = path
        for ch in node.get("children") or []:
            walk(ch, (path + " " + ch["name"]).strip())

    for g in doc["groups"]:
        walk(g, g["name"])
    for r in doc["recipes"]:
        walk(r, "recipe " + r["name"])
        if r["name"] not in explain:
            explain[r["name"]] = "recipe " + r["name"]
    for ex in doc["explain"]:
        if ex["name"] in explain:
            continue
        explain[ex["name"]] = ex.get("target") or "skip"
    if len(explain) < 164:
        raise SystemExit(f"explain count {len(explain)} < 164")
    print(f"groups={len(doc['groups'])} explain={len(explain)} recipes={len(doc['recipes'])}", file=sys.stderr)


def main() -> None:
    doc = build()
    validate(doc)
    OUT.parent.mkdir(parents=True, exist_ok=True)
    text = yaml.dump(doc, Dumper=Dumper, sort_keys=False, allow_unicode=True, width=1000)
    header = (
        "# Generated by scripts/gen_catalog.py. Edit the generator, then re-run it.\n"
        "# Bhuppi embeds this file. Python is not required to build or run bhuppi.\n"
    )
    OUT.write_text(header + text, encoding="utf-8")
    print(f"wrote {OUT}", file=sys.stderr)


if __name__ == "__main__":
    main()
