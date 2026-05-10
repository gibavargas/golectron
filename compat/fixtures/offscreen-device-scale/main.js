const { app, BrowserWindow } = require('electron')

app.whenReady().then(() => {
  const report = {
    defaultAccepted: false,
    defaultDeviceScaleFactor: 1,
    customAccepted: false,
    customDeviceScaleFactor: 2
  }

  try {
    const defaultWindow = new BrowserWindow({
      show: false,
      webPreferences: {
        offscreen: true
      }
    })
    report.defaultAccepted = true
    defaultWindow.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  try {
    const customWindow = new BrowserWindow({
      show: false,
      webPreferences: {
        offscreen: {
          deviceScaleFactor: 2
        }
      }
    })
    report.customAccepted = true
    customWindow.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
