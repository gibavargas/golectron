const { app, Notification } = require('electron')

app.whenReady().then(() => {
  const supported = process.platform === 'darwin' || process.platform === 'win32'
  const report = {
    platform: process.platform,
    supported,
    historyAvailable: typeof Notification.getHistory === 'function',
    removeFromHistoryAvailable: typeof Notification.removeFromHistory === 'function'
  }

  if (supported) {
    try {
      const options = {
        title: 'Deploy complete',
        body: 'Artifacts are ready',
        id: 'deploy-42',
        groupId: 'deployments'
      }
      if (process.platform === 'win32') {
        options.groupTitle = 'Deployments'
      }

      const notification = new Notification(options)
      report.id = notification.id
      report.groupId = notification.groupId
      report.groupTitle = notification.groupTitle || ''
      report.title = notification.title
      report.body = notification.body
    } catch (error) {
      report.error = error && error.message ? error.message : String(error)
    }
  }

  console.log(JSON.stringify(report))
  app.quit()
})
