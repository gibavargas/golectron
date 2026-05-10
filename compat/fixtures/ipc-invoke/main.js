const { app, BrowserWindow, ipcMain } = require('electron')
const path = require('node:path')

app.whenReady().then(() => {
  const report = {
    invokePong: false,
    argsEcho: false,
    onceFirst: false,
    onceSecondRejected: false,
    removedRejected: false,
    duplicateRejected: false,
    missingRejected: false
  }

  ipcMain.handle('fixture:ping', (_event, value) => {
    report.argsEcho = value === 'renderer'
    return 'pong'
  })
  try {
    ipcMain.handle('fixture:ping', () => 'duplicate')
  } catch (_error) {
    report.duplicateRejected = true
  }
  let onceCalls = 0
  ipcMain.handleOnce('fixture:once', () => {
    onceCalls += 1
    return onceCalls
  })
  ipcMain.handle('fixture:gone', () => 'gone')
  ipcMain.removeHandler('fixture:gone')
  ipcMain.once('fixture:done', (_event, rendererReport) => {
    Object.assign(report, rendererReport)
    console.log(JSON.stringify(report))
    app.quit()
  })

  const win = new BrowserWindow({
    show: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js')
    }
  })
  win.loadURL('data:text/html,<html><body>ipc</body></html>')
})
