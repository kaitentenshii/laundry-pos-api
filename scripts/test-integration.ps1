$ErrorActionPreference = 'Stop'
if (-not $env:TEST_DATABASE_URL) { throw 'Set TEST_DATABASE_URL to a dedicated migrated QA PostgreSQL database.' }
$goCommand = Get-Command go -ErrorAction SilentlyContinue
$goBinary = if ($goCommand) { $goCommand.Source } else { Join-Path $PSScriptRoot '../.cache/tools/go/bin/go.exe' }
Push-Location (Join-Path $PSScriptRoot '..')
try {
    & $goBinary test -count=1 ./...
    if ($LASTEXITCODE -ne 0) { throw 'Integration tests failed.' }
    & $goBinary vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }
} finally { Pop-Location }
