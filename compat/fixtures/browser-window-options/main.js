const { app, BrowserWindow } = require('electron')

app.whenReady().then(() => {
  const report = {
    width: 0,
    height: 0,
    visible: true,
    title: '',
    devTools: true,
    defaultDevTools: false,
    defaultContextIsolation: false,
    explicitContextIsolation: true,
    explicitNodeIntegration: false,
    modalConstructed: false
  }

  try {
    const defaults = new BrowserWindow({ show: false })
    const defaultPrefs = defaults.webContents.getLastWebPreferences()
    report.defaultDevTools = true
    report.defaultContextIsolation = defaultPrefs.contextIsolation
    defaults.destroy()

    const parent = new BrowserWindow({ show: false })
    const window = new BrowserWindow({
      width: 640,
      height: 480,
      show: false,
      title: 'Fixture',
      parent,
      modal: true,
      webPreferences: {
        devTools: false,
        contextIsolation: false,
        nodeIntegration: true
      }
    })
    const bounds = window.getBounds()
    const prefs = window.webContents.getLastWebPreferences()
    report.width = bounds.width
    report.height = bounds.height
    report.visible = window.isVisible()
    report.title = window.getTitle()
    report.devTools = false
    report.explicitContextIsolation = prefs.contextIsolation
    report.explicitNodeIntegration = prefs.nodeIntegration
    report.modalConstructed = window.isModal()
    window.destroy()
    parent.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
