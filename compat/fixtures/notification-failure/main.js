const { app, Notification } = require('electron')

app.whenReady().then(() => {
  const report = {
    platform: process.platform,
    unsupported: process.platform !== 'darwin',
    failed: false,
    shown: false,
    errorDomain: false
  }

  if (process.platform !== 'darwin') {
    console.log(JSON.stringify(report))
    app.quit()
    return
  }

  try {
    const notification = new Notification({
      title: 'Probe',
      body: 'Unsigned check'
    })
    const timer = setTimeout(() => {
      console.log(JSON.stringify(report))
      app.quit()
    }, 1500)
    const finish = () => {
      clearTimeout(timer)
      console.log(JSON.stringify(report))
      app.quit()
    }
    notification.once('failed', (_event, error) => {
      report.failed = true
      report.errorDomain = String(error || '').includes('UNErrorDomain')
      finish()
    })
    notification.once('show', () => {
      report.shown = true
      finish()
    })
    notification.show()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
    console.log(JSON.stringify(report))
    app.quit()
  }
})
