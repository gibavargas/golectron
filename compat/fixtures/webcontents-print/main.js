const { app, BrowserWindow } = require('electron')

function onceLoaded (contents) {
  return new Promise((resolve, reject) => {
    contents.once('did-finish-load', resolve)
    contents.once('did-fail-load', (_event, code, description) => {
      reject(new Error(`${code}: ${description}`))
    })
  })
}

app.whenReady().then(async () => {
  const report = {
    defaultAccepted: false,
    defaultPageSize: false,
    pageSizeOmitted: true,
    copiesDefault: 1,
    conflictRejected: false
  }

  try {
    const win = new BrowserWindow({ show: false })
    const loaded = onceLoaded(win.webContents)
    win.loadURL('data:text/html,<html><body>print</body></html>')
    await loaded

    try {
      win.webContents.print({
        silent: true,
        usePrinterDefaultPageSize: true
      }, () => {})
      report.defaultAccepted = true
      report.defaultPageSize = true
    } catch (_error) {
      report.defaultAccepted = false
    }

    try {
      win.webContents.print({
        silent: true,
        usePrinterDefaultPageSize: true,
        pageSize: 'A4'
      }, () => {})
    } catch (_error) {
      report.conflictRejected = true
    }

    win.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
