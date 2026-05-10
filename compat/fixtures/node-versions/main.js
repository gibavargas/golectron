const { app } = require('electron')

app.whenReady().then(() => {
  const report = {
    electron: process.versions.electron,
    chrome: process.versions.chrome,
    node: process.versions.node,
    v8: process.versions.v8,
    modules: process.versions.modules
  }

  console.log(JSON.stringify(report))
  app.quit()
})
