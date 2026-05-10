const { app, session } = require('electron')

app.whenReady().then(async () => {
  const report = {
    acceptsQuotas: false,
    errorName: ''
  }

  try {
    await session.defaultSession.clearStorageData({
      quotas: ['temporary']
    })
    report.acceptsQuotas = true
  } catch (error) {
    report.errorName = error && error.name ? error.name : ''
  }

  console.log(JSON.stringify(report))
  app.quit()
})
