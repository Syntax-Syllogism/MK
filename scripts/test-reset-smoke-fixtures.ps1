$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$target = Join-Path ([IO.Path]::GetTempPath()) ('mk-fixtures-' + [guid]::NewGuid())
try {
    & (Join-Path $PSScriptRoot 'reset-smoke-fixtures.ps1') -DocsDir $target | Out-Null
    $source = @(Get-ChildItem (Join-Path $root 'test-docs') -Filter '*.md' -File | Sort-Object Name)
    $copied = @(Get-ChildItem $target -Filter '*.md' -File | Sort-Object Name)
    if ($source.Count -ne $copied.Count) { throw "fixture count mismatch" }
    for ($i = 0; $i -lt $source.Count; $i++) {
        if ($source[$i].Name -ne $copied[$i].Name -or
            (Get-FileHash $source[$i].FullName).Hash -ne (Get-FileHash $copied[$i].FullName).Hash) {
            throw "fixture mismatch: $($source[$i].Name)"
        }
    }
    Write-Output "Reset smoke-fixture test passed ($($source.Count) fixtures)."
} finally {
    Remove-Item $target -Recurse -Force -ErrorAction SilentlyContinue
}
