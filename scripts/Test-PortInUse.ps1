param([int]$Port = 8000)
try {
    $c = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    return ($null -ne $c)
} catch {
    $out = netstat -ano | Select-String ":\s*$Port\s+.*LISTENING"
    return ($null -ne $out)
}
