const { app, globalShortcut } = require('electron')

app.whenReady().then(() => {
  const report = {
    initialSuspended: false,
    suspendedAfterSet: false,
    resumedAfterUnset: false
  }

  try {
    report.initialSuspended = globalShortcut.isSuspended()
    globalShortcut.setSuspended(true)
    report.suspendedAfterSet = globalShortcut.isSuspended()
    globalShortcut.setSuspended(false)
    report.resumedAfterUnset = !globalShortcut.isSuspended()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
