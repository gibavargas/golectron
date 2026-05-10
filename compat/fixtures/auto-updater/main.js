const { app, autoUpdater } = require('electron')

app.whenReady().then(() => {
  const supported = process.platform === 'darwin' || process.platform === 'win32'
  const report = {
    platform: process.platform,
    supported,
    feedURL: '',
    defaultProvider: 'squirrel'
  }

  if (supported) {
    try {
      autoUpdater.setFeedURL({ url: 'https://updates.example.test/feed' })
      report.feedURL = autoUpdater.getFeedURL()
    } catch (error) {
      report.error = error && error.message ? error.message : String(error)
    }
  }

  console.log(JSON.stringify(report))
  app.quit()
})
