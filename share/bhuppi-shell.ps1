# bhuppi PowerShell hook. Dot-source from $PROFILE.
# folders go / files go call Set-Location. Other builtins are not injected.

function bhuppi {
  $a = @($args)
  $dry = $false
  $help = $false
  $i = 0
  while ($i -lt $a.Count) {
    switch ($a[$i]) {
      '--dry-run' { $dry = $true; $i++; continue }
      '--plain' { $i++; continue }
      '--run' { $i++; continue }
      '-h' { $help = $true; $i++; continue }
      '--help' { $help = $true; $i++; continue }
      '--os' { $i += 2; continue }
      default { break }
    }
  }
  if (-not $dry -and -not $help -and ($i + 2) -lt $a.Count) {
    if (($a[$i] -eq 'files' -or $a[$i] -eq 'folders') -and $a[$i + 1] -eq 'go') {
      Set-Location -LiteralPath $a[$i + 2]
      return
    }
  }
  $exe = Join-Path $env:LOCALAPPDATA 'bhuppi\lib\bhuppi.exe'
  if (-not (Test-Path $exe)) { $exe = 'bhuppi.exe' }
  & $exe @args
}
