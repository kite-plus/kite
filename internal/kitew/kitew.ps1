# kitew.ps1 runs the Kite release this site pins in kite.lock, as kitew does
# on Linux and macOS, so that every machine and every deploy builds it with
# the same Kite. The first time, it downloads the release for this machine
# and checks it against the release's checksums, which kite.lock pins in
# turn; after that it runs the copy kept in the user's cache. `kite wrapper`
# writes this file and moves the pin.
#
# Where scripts may not run:
#   powershell -ExecutionPolicy Bypass -File kitew.ps1 <command>
#
#   KITE_DOWNLOAD_URL  where releases are downloaded from, in place of
#                      https://github.com/kite-plus/kite/releases/download

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

# Pinned reads a key of the kite section of kite.lock.
function Pinned([string]$Lock, [string]$Key) {
    $inside = $false
    foreach ($line in Get-Content -LiteralPath $Lock) {
        if ($line -match '^kite:') { $inside = $true; continue }
        if ($line -match '^[^\s#]') { $inside = $false }
        if ($inside -and $line -match "^\s+$Key\s*:\s*[`"']?([^`"'\s#]+)") { return $Matches[1] }
    }
    return ''
}

function Fetch([string]$Url, [string]$Out) {
    try {
        Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Out
    } catch {
        throw "could not download ${Url}: $($_.Exception.Message)"
    }
}

# Sha256 uses .NET rather than Get-FileHash: Windows PowerShell started from
# PowerShell 7 inherits a module path that hides its script modules, which
# Get-FileHash and Expand-Archive are part of.
function Sha256([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    try {
        $sha = [Security.Cryptography.SHA256]::Create()
        try { $hash = $sha.ComputeHash($stream) } finally { $sha.Dispose() }
    } finally {
        $stream.Dispose()
    }
    -join ($hash | ForEach-Object { $_.ToString('x2') })
}

try {
    $lock = Join-Path $PSScriptRoot 'kite.lock'
    if (-not (Test-Path -LiteralPath $lock)) {
        throw "there is no kite.lock beside kitew.ps1; run 'kite wrapper' to pin a Kite release"
    }
    $version = Pinned $lock 'version'
    $checksums = Pinned $lock 'checksums'
    if (-not $version) { throw "kite.lock pins no Kite release; run 'kite wrapper' to pin one" }

    if ($env:OS -eq 'Windows_NT') {
        $os = 'windows'
        # A 32-bit PowerShell on 64-bit Windows names the machine here.
        $machine = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
        $arch = switch ($machine) { 'AMD64' { 'amd64' } 'ARM64' { 'arm64' } default { '' } }
        $cacheRoot = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { [Environment]::GetFolderPath('LocalApplicationData') }
        $name = 'kite.exe'
        $format = 'zip'
    } else {
        $os = if ($IsMacOS) { 'darwin' } else { 'linux' }
        $machine = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
        $arch = switch ($machine) { 'X64' { 'amd64' } 'Arm64' { 'arm64' } 'Arm' { 'armv7' } default { '' } }
        $cacheRoot = if ($IsMacOS) { Join-Path $HOME 'Library/Caches' } elseif ($env:XDG_CACHE_HOME) { $env:XDG_CACHE_HOME } else { Join-Path $HOME '.cache' }
        $name = 'kite'
        $format = 'tar.gz'
    }
    if (-not $arch) { throw "there is no Kite release for $machine" }

    $cache = [IO.Path]::Combine($cacheRoot, 'kite', 'releases', $version)
    $kite = Join-Path $cache $name

    if (-not (Test-Path -LiteralPath $kite)) {
        $base = if ($env:KITE_DOWNLOAD_URL) { $env:KITE_DOWNLOAD_URL.TrimEnd('/') } else { 'https://github.com/kite-plus/kite/releases/download' }
        $archive = "kite_${version}_${os}_$arch.$format"
        # Inside the cache, so that the finished binary is moved into place
        # whole, however many builds start at once.
        $tmp = Join-Path $cache ('download.' + [Guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Force -Path $tmp | Out-Null
        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
            [Console]::Error.WriteLine("kitew: downloading Kite $version for $os/$arch")
            $list = Join-Path $tmp 'checksums.txt'
            Fetch "$base/v$version/checksums.txt" $list
            if ($checksums -and ('sha256:' + (Sha256 $list)) -ne $checksums) {
                throw "the checksums Kite $version is downloaded with are not the ones kite.lock pins"
            }
            $want = ''
            foreach ($line in Get-Content -LiteralPath $list) {
                $fields = -split $line
                if ($fields.Count -ge 2 -and ($fields[1] -eq $archive -or $fields[1] -eq "*$archive")) { $want = $fields[0].ToLowerInvariant() }
            }
            if (-not $want) { throw "the checksums of Kite $version list no $archive" }
            $file = Join-Path $tmp $archive
            Fetch "$base/v$version/$archive" $file
            if ((Sha256 $file) -ne $want) { throw "$archive does not match its checksum" }
            $out = Join-Path $tmp 'out'
            New-Item -ItemType Directory -Path $out | Out-Null
            if ($format -eq 'zip') {
                Add-Type -AssemblyName System.IO.Compression.FileSystem
                [IO.Compression.ZipFile]::ExtractToDirectory($file, $out)
            } else {
                tar -xzf $file -C $out kite
                if ($LASTEXITCODE -ne 0) { throw "could not unpack $archive" }
            }
            try {
                Move-Item -Force -LiteralPath (Join-Path $out $name) -Destination $kite
            } catch {
                # Another build finished first, and its copy may be running.
                if (-not (Test-Path -LiteralPath $kite)) { throw }
            }
        } finally {
            Remove-Item -Recurse -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
        }
    }
} catch {
    [Console]::Error.WriteLine("kitew: $($_.Exception.Message)")
    exit 1
}

& $kite @args
exit $LASTEXITCODE
