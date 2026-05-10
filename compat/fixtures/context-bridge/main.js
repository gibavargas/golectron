const { app, BrowserWindow } = require('electron')
const path = require('node:path')

app.whenReady().then(async () => {
  const report = {
    exposedVersion: false,
    functionCallable: false,
    arrayValueCopied: false,
    duplicateRejected: false,
    mutationRejected: false
  }

  try {
    const win = new BrowserWindow({
      show: false,
      webPreferences: {
        contextIsolation: true,
        preload: path.join(__dirname, 'preload.js')
      }
    })
    await win.loadURL('data:text/html,<html><body>context</body></html>')
    const rendererReport = await win.webContents.executeJavaScript(`(() => {
      const before = window.fixture.version
      window.fixture.version = 'mutated'
      return {
        exposedVersion: before === '1.0.0',
        functionCallable: window.fixture.add(2, 3) === 5,
        arrayValueCopied: Array.isArray(window.fixture.flags) && window.fixture.flags[0] === true,
        duplicateRejected: window.fixture.duplicateRejected() === true,
        mutationRejected: window.fixture.version === '1.0.0'
      }
    })()`)
    Object.assign(report, rendererReport)
    win.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
