if (-not (Test-Path C:\tools\datadog-package.exe)) {
    Write-Host "Downloading datadog-package.exe"
    (New-Object System.Net.WebClient).DownloadFile("https://dd-agent-omnibus.s3.amazonaws.com/datadog-package.exe", "C:\\tools\\datadog-package.exe")
}
$rawAgentVersion = "{0}-1" -f (inv agent.version --url-safe --major-version 7)
Write-Host "Detected agent version ${rawAgentVersion}"

$packageName = "datadog-agent-${rawAgentVersion}-windows-amd64.tar"

if (Test-Path .\omnibus\pkg\$packageName) {
    Remove-Item .\omnibus\pkg\$packageName
}

# The argument --archive-path ".\omnibus\pkg\datadog-agent-${version}.tar.gz" is currently broken and has no effects
& C:\tools\datadog-package.exe create --package datadog-agent --os windows --arch amd64 --archive --version $rawAgentVersion .\omnibus\pkg\
Copy-Item datadog-agent-${rawAgentVersion}-windows-amd64.tar .\omnibus\pkg\
