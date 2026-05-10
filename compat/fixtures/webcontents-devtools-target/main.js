const { app, BrowserWindow } = require('electron')

app.whenReady().then(() => {
  const report = {
    methodAvailable: false,
    firstNonempty: false,
    stable: false,
    distinct: false,
    lookupable: true
  }

  try {
    const first = new BrowserWindow({ show: false })
    const second = new BrowserWindow({ show: false })
    const firstMethod = first.webContents.getOrCreateDevToolsTargetId
    const secondMethod = second.webContents.getOrCreateDevToolsTargetId
    report.methodAvailable = typeof firstMethod === 'function' && typeof secondMethod === 'function'
    if (report.methodAvailable) {
      const firstId = first.webContents.getOrCreateDevToolsTargetId()
      const secondId = first.webContents.getOrCreateDevToolsTargetId()
      const otherId = second.webContents.getOrCreateDevToolsTargetId()
      report.firstNonempty = typeof firstId === 'string' && firstId.length > 0
      report.stable = firstId === secondId
      report.distinct = firstId !== otherId
    }
    first.destroy()
    second.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
