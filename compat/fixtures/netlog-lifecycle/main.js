const { app, netLog } = require('electron')
const path = require('node:path')

app.whenReady().then(async () => {
  const report = {
    started: false,
    activeDuring: false,
    stopped: false,
    inactiveAfter: false
  }

  try {
    const logPath = path.join(app.getPath('temp'), `electron-go-netlog-${process.pid}.json`)
    await netLog.startLogging(logPath)
    report.started = true
    report.activeDuring = netLog.currentlyLogging
    await netLog.stopLogging()
    report.stopped = true
    report.inactiveAfter = !netLog.currentlyLogging
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
