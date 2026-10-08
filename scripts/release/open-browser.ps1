# Waits until the server is actually accepting connections and only then opens
# the browser. start.bat runs this in the background, so opening the page
# immediately used to race the server start and show a connection error.
#
# Written for Windows PowerShell 5.1 (no ternary / null-coalescing) because
# that is what a fresh Windows install provides.
param(
    [int]$Port = 8000,
    [int]$TimeoutSeconds = 60
)

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
$ready = $false

while ((Get-Date) -lt $deadline) {
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect('127.0.0.1', $Port)
        $client.Close()
        $ready = $true
        break
    } catch {
        Start-Sleep -Milliseconds 400
    }
}

if ($ready) {
    Start-Process ("http://127.0.0.1:$Port/#/checkout")
} else {
    Write-Host "Server did not answer on port $Port within $TimeoutSeconds s." -ForegroundColor Red
    Write-Host "Check the server window for an error message." -ForegroundColor Red
    Read-Host "Press Enter to close"
}
