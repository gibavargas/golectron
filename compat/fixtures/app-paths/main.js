const { app } = require('electron')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

app.whenReady().then(() => {
  const report = {
    appPathNonempty: false,
    homePathNonempty: false,
    appDataPathNonempty: false,
    userDataPathNonempty: false,
    tempPathNonempty: false,
    exePathNonempty: false,
    modulePathNonempty: false,
    desktopPathNonempty: false,
    documentsPathNonempty: false,
    downloadsPathNonempty: false,
    musicPathNonempty: false,
    picturesPathNonempty: false,
    videosPathNonempty: false,
    crashDumpsPathNonempty: false,
    sessionDataDefaultsUserData: false,
    sessionDataOverrideWorks: false,
    logsOverrideWorks: false
  }

  try {
    const userData = app.getPath('userData')
    const sessionData = app.getPath('sessionData')
    const sessionOverride = fs.mkdtempSync(path.join(os.tmpdir(), 'electron-go-session-'))
    const logsOverride = fs.mkdtempSync(path.join(os.tmpdir(), 'electron-go-logs-'))

    app.setPath('sessionData', sessionOverride)
    app.setPath('logs', logsOverride)

    report.appPathNonempty = Boolean(app.getAppPath())
    report.homePathNonempty = Boolean(app.getPath('home'))
    report.appDataPathNonempty = Boolean(app.getPath('appData'))
    report.userDataPathNonempty = Boolean(userData)
    report.tempPathNonempty = Boolean(app.getPath('temp'))
    report.exePathNonempty = Boolean(app.getPath('exe'))
    report.modulePathNonempty = Boolean(app.getPath('module'))
    report.desktopPathNonempty = Boolean(app.getPath('desktop'))
    report.documentsPathNonempty = Boolean(app.getPath('documents'))
    report.downloadsPathNonempty = Boolean(app.getPath('downloads'))
    report.musicPathNonempty = Boolean(app.getPath('music'))
    report.picturesPathNonempty = Boolean(app.getPath('pictures'))
    report.videosPathNonempty = Boolean(app.getPath('videos'))
    report.crashDumpsPathNonempty = Boolean(app.getPath('crashDumps'))
    report.sessionDataDefaultsUserData = sessionData === userData
    report.sessionDataOverrideWorks = app.getPath('sessionData') === sessionOverride
    report.logsOverrideWorks = app.getPath('logs') === logsOverride
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
