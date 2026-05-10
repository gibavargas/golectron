const { app, nativeTheme, powerMonitor, screen, systemPreferences } = require('electron')

function goPlatformName(platform) {
  if (platform === 'win32') return 'windows'
  return platform
}

app.whenReady().then(() => {
  const platform = goPlatformName(process.platform)
  const primaryDisplay = screen.getPrimaryDisplay()
  const idleState = powerMonitor.getSystemIdleState(1)
  const idleTime = powerMonitor.getSystemIdleTime()
  const report = {
    platform,
    supportsNativeThemeCore: typeof nativeTheme.themeSource === 'string' &&
      typeof nativeTheme.shouldUseDarkColors === 'boolean',
    shouldDifferentiateWithoutColor: Boolean(nativeTheme.shouldDifferentiateWithoutColor),
    supportsDifferentiateWithoutColor: platform === 'darwin',
    screenPrimaryDisplayAvailable: Boolean(primaryDisplay && primaryDisplay.bounds &&
      primaryDisplay.bounds.width > 0 && primaryDisplay.bounds.height > 0),
    screenScaleFactorPositive: Boolean(primaryDisplay && primaryDisplay.scaleFactor > 0),
    powerMonitorIdleStateAvailable: ['active', 'idle', 'locked', 'unknown'].includes(idleState),
    powerMonitorIdleTimeNonNegative: typeof idleTime === 'number' && idleTime >= 0,
    systemPreferencesAvailable: platform === 'darwin' ?
      typeof systemPreferences.getUserDefault === 'function' :
      platform === 'windows'
  }

  if (!report.supportsDifferentiateWithoutColor) {
    report.shouldDifferentiateWithoutColor = false
  }

  console.log(JSON.stringify(report))
  app.quit()
})
