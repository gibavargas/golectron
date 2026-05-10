const { app, BrowserWindow } = require('electron')

app.whenReady().then(() => {
  const report = {
    defaultAccepted: false,
    defaultFocuses: true,
    explicitFalseAccepted: false,
    explicitFalseFocuses: false
  }

  try {
    const defaultWindow = new BrowserWindow({ show: false })
    report.defaultAccepted = true
    defaultWindow.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  try {
    const explicitWindow = new BrowserWindow({
      show: false,
      webPreferences: {
        focusOnNavigation: false
      }
    })
    report.explicitFalseAccepted = true
    explicitWindow.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
