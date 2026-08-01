function Find-PythonExe {
    $candidates = @()

    if (Get-Command py -ErrorAction SilentlyContinue) {
        foreach ($v in @("-3.12", "-3.13", "-3.11", "-3")) {
            try {
                $ver = & py $v -c "import sys; print(sys.executable)" 2>$null
                if ($ver -and (Test-Path $ver.Trim())) { $candidates += $ver.Trim() }
            } catch {}
        }
    }

    if (Get-Command python -ErrorAction SilentlyContinue) {
        $cmd = (Get-Command python).Source
        if ($cmd -notmatch "WindowsApps") { $candidates += $cmd }
    }

    $roots = @(
        "$env:LOCALAPPDATA\Programs\Python",
        "$env:LOCALAPPDATA\Python",
        "C:\Program Files\Python312",
        "C:\Program Files\Python313",
        "C:\Program Files\Python311"
    )
    foreach ($root in $roots) {
        if (Test-Path $root) {
            Get-ChildItem $root -Filter "python.exe" -Recurse -Depth 4 -ErrorAction SilentlyContinue |
                ForEach-Object { $candidates += $_.FullName }
        }
    }

    $valid = @()
    foreach ($exe in ($candidates | Select-Object -Unique)) {
        try {
            $vi = & $exe -c "import sys; print(sys.version_info[0], sys.version_info[1])" 2>$null
            if ($LASTEXITCODE -ne 0 -or -not $vi) { continue }
            $parts = $vi.Trim() -split '\s+'
            $major = [int]$parts[0]
            $minor = [int]$parts[1]
            if ($major -eq 3 -and $minor -ge 11) {
                $valid += [PSCustomObject]@{ Path = $exe; Minor = $minor }
            }
        } catch {}
    }

    if ($valid.Count -eq 0) { return $null }

    # Prefer 3.12 / 3.13 over 3.14 (wheels)
    $best = $valid | Where-Object { $_.Minor -le 13 } | Sort-Object Minor -Descending | Select-Object -First 1
    if (-not $best) { $best = $valid | Sort-Object Minor -Descending | Select-Object -First 1 }
    return $best.Path
}
