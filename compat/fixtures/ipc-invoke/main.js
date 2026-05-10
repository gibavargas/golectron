const { app, BrowserWindow, MessageChannelMain, ipcMain } = require('electron')
const path = require('node:path')

function waitForMainMessagePort () {
  return new Promise((resolve) => {
    const { port1, port2 } = new MessageChannelMain()
    port1.once('message', (event) => {
      port1.close()
      port2.close()
      resolve(event.data === 'from-port2')
    })
    port2.postMessage('from-port2')
    port1.start()
  })
}

app.whenReady().then(async () => {
  const report = {
    invokePong: false,
    argsEcho: false,
    onceFirst: false,
    onceSecondRejected: false,
    removedRejected: false,
    duplicateRejected: false,
    missingRejected: false,
    messagePortRoundTrip: await waitForMainMessagePort(),
    transferredPortMessage: false
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
  ipcMain.once('fixture:renderer-port', (event) => {
    const [port] = event.ports
    port.once('message', (messageEvent) => {
      report.transferredPortMessage = messageEvent.data === 'from-renderer-port'
      port.close()
    })
    port.start()
  })
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
