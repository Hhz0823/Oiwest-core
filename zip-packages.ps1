$BUILD_DIR = "d:\Oiwest core\build"
$VERSION = "2.1.0"

Remove-Item -Recurse -Force "$BUILD_DIR\zip" -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force "$BUILD_DIR\zip" | Out-Null

$dirs = Get-ChildItem $BUILD_DIR -Directory | Where-Object { $_.Name -ne "zip" }

foreach ($d in $dirs) {
    $files = Get-ChildItem $d.FullName -File
    if ($files.Count -eq 0) { continue }

    $zipName = "oiwest-core-v${VERSION}-$($d.Name).zip"
    $zipPath = Join-Path "$BUILD_DIR\zip" $zipName

    Write-Host "Packaging: $zipName"
    Compress-Archive -Path ($d.FullName + "\*") -DestinationPath $zipPath -Force
    $size = [math]::Round((Get-Item $zipPath).Length / 1MB, 2)
    Write-Host "  Size: $size MB"
}

Write-Host ""
Write-Host "=== ZIP packages ==="
Get-ChildItem "$BUILD_DIR\zip" | ForEach-Object {
    $s = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $($_.Name)  ${s}MB"
}
Write-Host ""
Write-Host "Output: $BUILD_DIR\zip"
Write-Host "Done!"

