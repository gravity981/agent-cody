$ErrorActionPreference = 'SilentlyContinue'
$payload = [Console]::In.ReadToEnd()
$OutputEncoding = New-Object System.Text.UTF8Encoding -ArgumentList $false

$targets = @(
    'http://127.0.0.1:8765/hooks/copilot',
    'http://host.docker.internal:8765/hooks/copilot',
    'http://172.17.0.1:8765/hooks/copilot'
)

foreach ($url in $targets) {
    $response = $payload | & curl.exe -fsS --connect-timeout 0.1 -m 0.1 -H 'Expect:' --data-binary '@-' $url 2>$null
    if ($LASTEXITCODE -eq 0 -and (($response -join '').Trim() -eq 'ok')) {
        break
    }
}
exit 0
