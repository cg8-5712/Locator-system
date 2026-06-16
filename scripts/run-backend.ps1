$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

New-Item -ItemType Directory -Force .cache\go-build, .cache\pkg\mod, .cache\go | Out-Null

$env:GOCACHE = (Resolve-Path ".cache\go-build")
$env:GOMODCACHE = (Resolve-Path ".cache\pkg\mod")
$env:GOPATH = (Resolve-Path ".cache\go")
$env:GOSUMDB = "off"

go run ./cmd/server
