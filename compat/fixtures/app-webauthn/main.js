const { app } = require('electron')

app.whenReady().then(() => {
  const report = {
    configured: false,
    touchIDSupported: process.platform === 'darwin',
    keychainGroupStored: false,
    invalidRejected: false
  }

  try {
    app.configureWebAuthn({
      touchID: {
        keychainAccessGroup: 'TEAM.bundle'
      }
    })
    report.configured = true
    report.keychainGroupStored = true
    try {
      app.configureWebAuthn({
        touchID: {
          keychainAccessGroup: 'bad\ngroup'
        }
      })
    } catch (_error) {
      report.invalidRejected = true
    }
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
