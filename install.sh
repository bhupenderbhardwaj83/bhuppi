#!/bin/sh
# Install bhuppi onto a directory already on PATH, then hook bash and zsh.
# Usage: ./install.sh
#        ./install.sh --go-decision    # prints use|fetch and does not download
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
cd "$ROOT"

GO_VER=1.22.12

go_major() {
  bin=$1
  ver=$("$bin" env GOVERSION 2>/dev/null || true)
  case "$ver" in
    go1.*) ;;
    *)
      ver=$("$bin" version 2>/dev/null || true)
      ver=${ver#"go version "}
      ver=${ver%% *}
      ;;
  esac
  case "$ver" in
    go1.*)
      rest=${ver#go1.}
      printf '%s\n' "${rest%%.*}"
      ;;
  esac
}

go_ok() {
  bin=$1
  [ -n "$bin" ] || return 1
  [ -x "$bin" ] || return 1
  major=$(go_major "$bin")
  case "$major" in
    ''|*[!0-9]*) return 1 ;;
  esac
  [ "$major" -ge 22 ]
}

pick_go() {
  if command -v go >/dev/null 2>&1; then
    found=$(command -v go)
    if go_ok "$found"; then
      printf '%s\n' "$found"
      return 0
    fi
  fi
  for c in \
    "$HOME/.local/go/bin/go" \
    "$ROOT/.toolchain/go/bin/go" \
    /usr/local/go/bin/go \
    /opt/homebrew/bin/go \
    /opt/homebrew/opt/go/bin/go
  do
    if go_ok "$c"; then
      printf '%s\n' "$c"
      return 0
    fi
  done
  return 1
}

go_decision() {
  if pick_go >/dev/null 2>&1; then
    printf '%s\n' use
    return 0
  fi
  printf '%s\n' fetch
}

if [ "${1:-}" = "--go-decision" ]; then
  go_decision
  exit 0
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "bhuppi install: need $1" >&2
    exit 1
  fi
}

fetch_url() {
  url=$1
  out=$2
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
    return
  fi
  return 1
}

ensure_curl() {
  if command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1; then
    return 0
  fi
  if command -v apt-get >/dev/null 2>&1; then
    echo "bhuppi install: installing curl and tar"
    if [ "$(id -u)" -eq 0 ]; then
      apt-get update && apt-get install -y curl ca-certificates tar
    elif command -v sudo >/dev/null 2>&1; then
      sudo apt-get update && sudo apt-get install -y curl ca-certificates tar
    fi
  fi
  if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    echo "bhuppi install: curl or wget is required to fetch Go" >&2
    exit 1
  fi
  if ! command -v tar >/dev/null 2>&1; then
    echo "bhuppi install: tar is required to unpack Go" >&2
    exit 1
  fi
}

fetch_go() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$os" in
    darwin|linux) ;;
    *)
      echo "bhuppi install: use install.cmd on Windows" >&2
      exit 1
      ;;
  esac
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *)
      echo "bhuppi install: unsupported arch $arch" >&2
      exit 1
      ;;
  esac
  url="https://go.dev/dl/go${GO_VER}.${os}-${arch}.tar.gz"
  tmp=$(mktemp -d)
  echo "bhuppi install: fetching $url"
  if ! fetch_url "$url" "$tmp/go.tgz"; then
    if [ "$os" = "darwin" ] && command -v brew >/dev/null 2>&1; then
      echo "bhuppi install: tarball failed; last resort is Homebrew"
      brew install go
      return 0
    fi
    echo "bhuppi install: could not fetch the Go ${GO_VER} tarball" >&2
    exit 1
  fi
  mkdir -p "$HOME/.local"
  rm -rf "$HOME/.local/go"
  tar -C "$HOME/.local" -xzf "$tmp/go.tgz"
  rm -rf "$tmp"
  PATH="$HOME/.local/go/bin:$PATH"
  export PATH
}

ensure_go() {
  got=$(pick_go || true)
  if [ -n "$got" ]; then
    PATH="$(dirname "$got"):$PATH"
    export PATH
    echo "bhuppi install: using $got ($("$got" env GOVERSION 2>/dev/null || "$got" version))"
    return 0
  fi
  ensure_curl
  fetch_go
  if [ "$(go_decision)" != "use" ]; then
    echo "bhuppi install: Go ${GO_VER} is not usable after fetch" >&2
    exit 1
  fi
  echo "bhuppi install: using $HOME/.local/go/bin/go"
}

toolchain_dir() {
  d=$1
  case "$d" in
    */.local/go/bin|*/.toolchain/go/bin|/usr/local/go/bin) return 0 ;;
  esac
  if [ -x "$d/go" ] && [ -x "$d/gofmt" ]; then
    return 0
  fi
  return 1
}

install_bin() {
  src=$1
  if [ -n "${BHUPPI_PREFIX:-}" ]; then
    mkdir -p "$BHUPPI_PREFIX/bin"
    cp "$src" "$BHUPPI_PREFIX/bin/bhuppi"
    chmod 755 "$BHUPPI_PREFIX/bin/bhuppi"
    printf '%s\n' "$BHUPPI_PREFIX/bin/bhuppi"
    return 0
  fi
  # /usr/local/bin is on the default macOS PATH. Never drop the only copy
  # into the Go toolchain directory that was prepended just for the build.
  if [ -d /usr/local/bin ]; then
    if [ -w /usr/local/bin ]; then
      cp "$src" /usr/local/bin/bhuppi
      chmod 755 /usr/local/bin/bhuppi
      printf '%s\n' /usr/local/bin/bhuppi
      return 0
    fi
    if command -v sudo >/dev/null 2>&1; then
      echo "bhuppi install: copying to /usr/local/bin" >&2
      if sudo -n cp "$src" /usr/local/bin/bhuppi && sudo -n chmod 755 /usr/local/bin/bhuppi; then
        printf '%s\n' /usr/local/bin/bhuppi
        return 0
      fi
      echo "bhuppi install: /usr/local/bin is not writable without a password; using ~/.local/bin" >&2
    fi
  fi
  if [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin"; then
    if [ -w "$HOME/.local/bin" ]; then
      cp "$src" "$HOME/.local/bin/bhuppi"
      chmod 755 "$HOME/.local/bin/bhuppi"
      printf '%s\n' "$HOME/.local/bin/bhuppi"
      return 0
    fi
  fi
  dest=""
  old=$IFS
  IFS=:
  for d in $PATH; do
    [ -n "$d" ] || continue
    case "$d" in
      /bin|/usr/bin|/sbin|/usr/sbin) continue ;;
    esac
    if toolchain_dir "$d"; then
      continue
    fi
    if [ -d "$d" ] && [ -w "$d" ]; then
      dest=$d
      break
    fi
  done
  IFS=$old
  if [ -z "$dest" ]; then
    mkdir -p "$HOME/.local/bin"
    dest=$HOME/.local/bin
  fi
  case "$dest/bhuppi" in
    /bin/ls|/usr/bin/ls) echo "bhuppi install: refusing to shadow ls" >&2; exit 1 ;;
  esac
  cp "$src" "$dest/bhuppi"
  chmod 755 "$dest/bhuppi"
  printf '%s\n' "$dest/bhuppi"
}

hook_rc() {
  file=$1
  script=$2
  marker="# >>> bhuppi >>>"
  if [ -f "$file" ] && grep -q "$marker" "$file"; then
    return 0
  fi
  touch "$file" || return 0
  printf '\n%s\n# files go / folders go, and tab completion\n[ -f "%s" ] && . "%s"\n# <<< bhuppi <<<\n' \
    "$marker" "$script" "$script" >> "$file"
}

# Comment out alias bhuppi='source ~/.bhuppi' without touching other lines.
retire_bhuppi_alias() {
  file=$1
  [ -f "$file" ] || return 0
  if ! grep -q 'alias bhuppi=' "$file"; then
    return 0
  fi
  tmp=$(mktemp)
  awk '
    /^[[:space:]]*alias[[:space:]]+bhuppi=/ && /\.bhuppi/ {
      print "# retired by bhuppi install — /usr/local/bin/bhuppi and ~/.config/bhuppi/zsh.sh"
      next
    }
    { print }
  ' "$file" > "$tmp"
  mv "$tmp" "$file"
}

install_shell() {
  bin=$1
  if [ "${BHUPPI_SKIP_HOOK:-}" = "1" ]; then
    return 0
  fi
  cfg=${BHUPPI_CONFIG_HOME:-"$HOME/.config/bhuppi"}
  mkdir -p "$cfg"
  {
    printf 'BHUPPI_BIN=%s\n' "'$bin'"
    "$bin" shell-init zsh
  } > "$cfg/zsh.sh"
  {
    printf 'BHUPPI_BIN=%s\n' "'$bin'"
    "$bin" shell-init bash
  } > "$cfg/bash.sh"
  cp "$ROOT/share/bhuppi-shell.sh" "$cfg/hook.sh"
  hook_rc "$HOME/.zshrc" "$cfg/zsh.sh"
  hook_rc "$HOME/.bashrc" "$cfg/bash.sh"
  if [ -f "$HOME/.bash_profile" ] || [ ! -f "$HOME/.bashrc" ]; then
    hook_rc "$HOME/.bash_profile" "$cfg/bash.sh"
  fi
  for rc in "$HOME/.zshrc" "$HOME/.zprofile" "$HOME/.zshenv" "$HOME/.bashrc" "$HOME/.bash_profile"; do
    retire_bhuppi_alias "$rc"
  done
  snip='# bhuppi — POSIX (dash-safe). The cd hook only.
if [ -n "${HOME-}" ] && [ -f "$HOME/.config/bhuppi/hook.sh" ]; then
  . "$HOME/.config/bhuppi/hook.sh"
fi
'
  if [ -n "${BHUPPI_CONFIG_HOME:-}" ]; then
    return 0
  fi
  if [ -d /etc/profile.d ] && [ -w /etc/profile.d ]; then
    printf '%s\n' "$snip" > /etc/profile.d/bhuppi.sh
  elif [ -d /etc/profile.d ] && command -v sudo >/dev/null 2>&1; then
    printf '%s\n' "$snip" | sudo -n tee /etc/profile.d/bhuppi.sh >/dev/null || true
  fi
}

ensure_go
need go
export CGO_ENABLED=0
mkdir -p "$ROOT/dist"
echo "bhuppi install: building"
go build -trimpath -ldflags "-s -w" -o "$ROOT/dist/bhuppi" ./cmd/bhuppi
installed=$(install_bin "$ROOT/dist/bhuppi")
if [ "$installed" != "$HOME/.local/go/bin/bhuppi" ] && [ -e "$HOME/.local/go/bin/bhuppi" ]; then
  rm -f "$HOME/.local/go/bin/bhuppi"
fi
install_shell "$installed"
echo "bhuppi install: $installed"
echo "try: bhuppi help"
