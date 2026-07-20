param(
  [string]$OutputPath = (Join-Path $PSScriptRoot "..\assets\social-preview.png")
)

Add-Type -AssemblyName System.Drawing

$width = 1200
$height = 630
$bitmap = [System.Drawing.Bitmap]::new($width, $height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit

try {
  $graphics.Clear([System.Drawing.ColorTranslator]::FromHtml("#0b0f12"))

  $gridPen = [System.Drawing.Pen]::new([System.Drawing.Color]::FromArgb(30, 120, 165, 182), 1)
  for ($x = 0; $x -le $width; $x += 48) { $graphics.DrawLine($gridPen, $x, 0, $x, $height) }
  for ($y = 0; $y -le $height; $y += 48) { $graphics.DrawLine($gridPen, 0, $y, $width, $y) }

  $mint = [System.Drawing.ColorTranslator]::FromHtml("#54e6b0")
  $paper = [System.Drawing.ColorTranslator]::FromHtml("#f4f7f5")
  $muted = [System.Drawing.ColorTranslator]::FromHtml("#9aa7a3")
  $linePen = [System.Drawing.Pen]::new([System.Drawing.Color]::FromArgb(85, $mint), 2)
  $nodeBrush = [System.Drawing.SolidBrush]::new($mint)

  $points = @(
    [System.Drawing.PointF]::new(760, 135), [System.Drawing.PointF]::new(925, 95),
    [System.Drawing.PointF]::new(1080, 185), [System.Drawing.PointF]::new(850, 265),
    [System.Drawing.PointF]::new(1035, 340), [System.Drawing.PointF]::new(750, 415),
    [System.Drawing.PointF]::new(950, 500), [System.Drawing.PointF]::new(1110, 470)
  )
  foreach ($pair in @(@(0,1),@(0,3),@(1,2),@(1,3),@(2,4),@(3,4),@(3,5),@(4,6),@(4,7),@(5,6),@(6,7))) {
    $graphics.DrawLine($linePen, $points[$pair[0]], $points[$pair[1]])
  }
  foreach ($point in $points) { $graphics.FillEllipse($nodeBrush, $point.X - 5, $point.Y - 5, 10, 10) }
  $graphics.FillEllipse([System.Drawing.SolidBrush]::new($paper), 842, 257, 16, 16)

  $iconPath = Join-Path $PSScriptRoot "..\assets\appicon.png"
  $icon = [System.Drawing.Image]::FromFile($iconPath)
  try { $graphics.DrawImage($icon, 82, 76, 94, 94) } finally { $icon.Dispose() }

  $titleFont = [System.Drawing.Font]::new("Segoe UI", 62, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
  $taglineFont = [System.Drawing.Font]::new("Segoe UI", 31, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
  $metaFont = [System.Drawing.Font]::new("Consolas", 17, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
  $paperBrush = [System.Drawing.SolidBrush]::new($paper)
  $mutedBrush = [System.Drawing.SolidBrush]::new($muted)
  try {
    $graphics.DrawString("Entcoin", $titleFont, $paperBrush, 82, 208)
    $graphics.DrawString("Run the network. Verify everything.", $taglineFont, $paperBrush, 86, 315)
    $graphics.DrawString("PROOF OF WORK / DESKTOP FULL NODE / OPEN SOURCE", $metaFont, $mutedBrush, 88, 397)
    $graphics.FillRectangle($nodeBrush, 88, 484, 72, 5)
    $graphics.DrawString("entcoin.xyz", $metaFont, $mutedBrush, 88, 514)
  } finally {
    $titleFont.Dispose()
    $taglineFont.Dispose()
    $metaFont.Dispose()
    $paperBrush.Dispose()
    $mutedBrush.Dispose()
  }

  $directory = Split-Path -Parent $OutputPath
  if (-not (Test-Path -LiteralPath $directory)) { New-Item -ItemType Directory -Path $directory | Out-Null }
  $bitmap.Save($OutputPath, [System.Drawing.Imaging.ImageFormat]::Png)
} finally {
  $graphics.Dispose()
  $bitmap.Dispose()
}
