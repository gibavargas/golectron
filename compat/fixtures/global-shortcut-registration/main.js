const { app, globalShortcut } = require('electron')

app.whenReady().then(() => {
  const accelerator = 'CommandOrControl+Alt+Shift+F19'
  const alias = 'CmdOrCtrl+Option+Shift+F19'
  const report = {
    registered: false,
    duplicateRejected: false,
    aliasRegistered: false,
    unregistered: false,
    unregisterAllCleared: false
  }

  try {
    report.registered = globalShortcut.register(accelerator, () => {})
    report.duplicateRejected = !globalShortcut.register(alias, () => {})
    report.aliasRegistered = globalShortcut.isRegistered(alias)
    globalShortcut.unregister(accelerator)
    report.unregistered = !globalShortcut.isRegistered(alias)
    globalShortcut.register(accelerator, () => {})
    globalShortcut.unregisterAll()
    report.unregisterAllCleared = !globalShortcut.isRegistered(accelerator)
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
