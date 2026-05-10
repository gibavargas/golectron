const { app, Notification } = require('electron')

app.whenReady().then(() => {
  const report = {
    title: '',
    body: '',
    silent: false,
    defaultUrgency: 'normal'
  }

  try {
    const notification = new Notification({
      title: 'Build complete',
      body: 'Artifacts are ready',
      silent: true
    })
    report.title = notification.title
    report.body = notification.body
    report.silent = notification.silent
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
