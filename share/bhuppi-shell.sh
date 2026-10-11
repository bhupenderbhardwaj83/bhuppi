# bhuppi shell hook. Source this from bash or zsh.
# files go / folders go cd the current shell. A child process cannot.
# The function calls the real binary. It does not source a dotfile.

# zsh expands an existing alias while it reads the next command.
# Remove that alias before the function below is defined.
unalias bhuppi 2>/dev/null || true
if [ -z "${BHUPPI_BIN:-}" ]; then
  if [ -x /usr/local/bin/bhuppi ]; then
    BHUPPI_BIN=/usr/local/bin/bhuppi
  else
    BHUPPI_BIN=$(command -v bhuppi 2>/dev/null || true)
  fi
fi

bhuppi() {
  _line=
  if [ -n "${BHUPPI_BIN:-}" ] && [ -x "$BHUPPI_BIN" ]; then
    _line=$("$BHUPPI_BIN" --print-cd "$@" 2>/dev/null) || _line=
  fi
  case "$_line" in
    cd\ *)
      cd -- "${_line#cd }" || return
      return
      ;;
  esac
  if [ -n "${BHUPPI_BIN:-}" ] && [ -x "$BHUPPI_BIN" ]; then
    "$BHUPPI_BIN" "$@"
    return
  fi
  command bhuppi "$@"
}
