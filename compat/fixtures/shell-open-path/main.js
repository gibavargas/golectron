const { app, shell } = require('electron')
const os = require('node:os')
const path = require('node:path')

app.whenReady().then(async () => {
  const report = {
    missingPathRejected: false,
    errorMessageNonempty: false
  }

  try {
    const missingPath = path.join(os.tmpdir(), 'electron-go-shell-open-path-missing')
    const errorMessage = await shell.openPath(missingPath)
    report.missingPathRejected = errorMessage !== ''
    report.errorMessageNonempty = errorMessage !== ''
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
