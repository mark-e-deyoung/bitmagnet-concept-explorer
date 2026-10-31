# Public-source, metadata-only native Windows PowerShell 5.1 RDTE qualification.
# Independent oracle for exact bencoded info span. No private Bitmagnet source copied here.
[CmdletBinding()]
param([string]$OutputDirectory = $env:RUNNER_TEMP)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0
if ($PSVersionTable.PSVersion.Major -ne 5 -or $PSVersionTable.PSVersion.Minor -ne 1) {
    throw 'This qualification must run under native Windows PowerShell 5.1.'
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    throw 'Output directory is required.'
}
[System.IO.Directory]::CreateDirectory($OutputDirectory) | Out-Null

function Skip-BencodedValue {
    param([byte[]]$Data, [hashtable]$Cursor, [int]$Depth)
    if ($Depth -gt 64 -or $Cursor.Index -ge $Data.Length) {
        throw 'Truncated or excessive nesting.'
    }
    $tag = [int]$Data[[int]$Cursor.Index]
    if ($tag -eq 105) {
        $Cursor.Index++
        $digits = 0
        while ($Cursor.Index -lt $Data.Length -and [int]$Data[[int]$Cursor.Index] -ne 101) {
            $digit = [int]$Data[[int]$Cursor.Index]
            if (($digit -lt 48 -or $digit -gt 57) -and -not ($digits -eq 0 -and $digit -eq 45)) {
                throw 'Invalid integer.'
            }
            $digits++
            $Cursor.Index++
        }
        if ($digits -eq 0 -or $Cursor.Index -ge $Data.Length) {
            throw 'Unterminated integer.'
        }
        $Cursor.Index++
        return
    }
    if ($tag -eq 108 -or $tag -eq 100) {
        $Cursor.Index++
        while ($Cursor.Index -lt $Data.Length -and [int]$Data[[int]$Cursor.Index] -ne 101) {
            Skip-BencodedValue -Data $Data -Cursor $Cursor -Depth ($Depth + 1)
        }
        if ($Cursor.Index -ge $Data.Length) { throw 'Unterminated list/dictionary.' }
        $Cursor.Index++
        return
    }
    if ($tag -lt 48 -or $tag -gt 57) { throw 'Invalid bencode string marker.' }
    $length = [long]0
    while ($Cursor.Index -lt $Data.Length) {
        $digit = [int]$Data[[int]$Cursor.Index]
        if ($digit -lt 48 -or $digit -gt 57) { break }
        $length = ($length * 10) + $digit - 48
        if ($length -gt $Data.Length) { throw 'String exceeds input limit.' }
        $Cursor.Index++
    }
    if ($Cursor.Index -ge $Data.Length -or [int]$Data[[int]$Cursor.Index] -ne 58) {
        throw 'Missing string colon.'
    }
    $Cursor.Index++
    if ($length -gt ([long]$Data.Length - [long]$Cursor.Index)) {
        throw 'String overruns input.'
    }
    $Cursor.Index += [int]$length
}

function Get-ExactRawInfoHash {
    param([byte[]]$Data)
    if ($Data.Length -lt 8 -or [int]$Data[0] -ne 100) {
        throw 'Torrent root must be a dictionary.'
    }
    $cursor = @{ Index = 1 }
    $start = -1
    $end = -1
    while ($cursor.Index -lt $Data.Length -and [int]$Data[[int]$cursor.Index] -ne 101) {
        $keyStart = [int]$cursor.Index
        Skip-BencodedValue -Data $Data -Cursor $cursor -Depth 1
        $isInfo = (($cursor.Index - $keyStart) -eq 6 -and
            [int]$Data[$keyStart] -eq 52 -and [int]$Data[$keyStart + 1] -eq 58 -and
            [int]$Data[$keyStart + 2] -eq 105 -and [int]$Data[$keyStart + 3] -eq 110 -and
            [int]$Data[$keyStart + 4] -eq 102 -and [int]$Data[$keyStart + 5] -eq 111)
        $valueStart = [int]$cursor.Index
        Skip-BencodedValue -Data $Data -Cursor $cursor -Depth 1
        if ($isInfo) {
            if ($start -ge 0) { throw 'Duplicate info dictionary.' }
            $start = $valueStart
            $end = [int]$cursor.Index
        }
    }
    if ($start -lt 0 -or $end -le $start -or $cursor.Index -ne ($Data.Length - 1)) {
        throw 'Missing info span or invalid root termination.'
    }
    $sha = [System.Security.Cryptography.SHA1]::Create()
    try {
        $bytes = $sha.ComputeHash($Data, $start, $end - $start)
        return [System.BitConverter]::ToString($bytes).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha.Dispose()
    }
}

[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
$cases = @(
    @{
        Name = 'ubuntu-26.04.1-desktop-amd64.iso.torrent'
        SHA256 = 'a4b59d67dfeffa07211b433cee5a0b1e545329b899be2c6cc83725cb13aafad6'
        InfoHash = '5b1e0d988fc7a0c9e99bd852071681a59974b39f'
    },
    @{
        Name = 'ubuntu-26.04.1-live-server-amd64.iso.torrent'
        SHA256 = '6487c84e88f05ac802d72251213ec0e619df1c806c0303c4fa6e8780251b8640'
        InfoHash = 'e4f19cf636409b72571766c2ae89c98fc21a109d'
    }
)
foreach ($case in $cases) {
    $url = 'https://releases.ubuntu.com/26.04/' + $case.Name
    $file = Join-Path $OutputDirectory $case.Name
    Invoke-WebRequest -Uri $url -UseBasicParsing -OutFile $file -TimeoutSec 45
    $sha256 = (Get-FileHash -Path $file -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($sha256 -cne $case.SHA256) { throw "Source SHA-256 mismatch: $($case.Name), got $sha256" }
    $original = [System.IO.File]::ReadAllBytes($file)
    $infohash = Get-ExactRawInfoHash -Data $original
    if ($infohash -cne $case.InfoHash) { throw "Raw infohash mismatch: $($case.Name), got $infohash" }
    Write-Host "REAL PS5.1 PASS: $($case.Name) sha256=$sha256 raw_infohash=$infohash"
}
$duplicate = [System.Text.Encoding]::ASCII.GetBytes('d4:infoi1e4:infoi2ee')
$rejected = $false
try { $null = Get-ExactRawInfoHash -Data $duplicate } catch { $rejected = $true }
if (-not $rejected) { throw 'Duplicate info key was accepted.' }
$malformed = [System.Text.Encoding]::ASCII.GetBytes('d4:info9:fooe')
$rejected = $false
try { $null = Get-ExactRawInfoHash -Data $malformed } catch { $rejected = $true }
if (-not $rejected) { throw 'Truncated source span was accepted.' }
Write-Host 'Native PS5.1 authentic-source identity + malformed/duplicate negative tests PASS.'
