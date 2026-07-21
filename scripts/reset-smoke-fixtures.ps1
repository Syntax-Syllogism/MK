param(
    [string]$DocsDir = (Join-Path $PSScriptRoot '..\docs')
)

$ErrorActionPreference = 'Stop'
$sourceDir = Join-Path $PSScriptRoot '..\test-docs'

if (-not (Test-Path -LiteralPath $sourceDir -PathType Container)) {
    throw "Fixture source directory does not exist: $sourceDir"
}

New-Item -ItemType Directory -Force -Path $DocsDir | Out-Null
$fixtures = Get-ChildItem -LiteralPath $sourceDir -Filter '*.md' -File
foreach ($fixture in $fixtures) {
    Copy-Item -LiteralPath $fixture.FullName -Destination (Join-Path $DocsDir $fixture.Name) -Force
    Write-Output "reset: $($fixture.Name)"
}

Write-Output ""
Write-Output "Reset $($fixtures.Count) fixture(s) in $DocsDir from $sourceDir."
Write-Output "test-docs/ remains the pristine source of truth."
