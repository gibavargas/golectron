const { app, nativeTheme } = require('electron')

function goPlatformName(platform) {
  if (platform === 'win32') return 'windows'
  return platform
}

app.whenReady().then(() => {
  const platform = goPlatformName(process.platform)
  const report = {
    platform,
    shouldDifferentiateWithoutColor: Boolean(nativeTheme.shouldDifferentiateWithoutColor),
    supportsDifferentiateWithoutColor: platform === 'darwin'
  }

  if (!report.supportsDifferentiateWithoutColor) {
    report.shouldDifferentiateWithoutColor = false
  }

  console.log(JSON.stringify(report))
  app.quit()
})
