const { app, crashReporter } = require('electron')

crashReporter.start({
  uploadToServer: false,
  extra: {
    channel: 'stable'
  }
})

app.whenReady().then(() => {
  const report = {
    started: true,
    uploadInitial: crashReporter.getUploadToServer(),
    uploadAfterSet: false,
    extraInitial: false,
    extraAdded: false,
    extraRemoved: false,
    uploadedReportsEmpty: false,
    lastReportNull: false
  }

  try {
    let parameters = crashReporter.getParameters()
    report.extraInitial = parameters.channel === 'stable'
    crashReporter.addExtraParameter('build', '42')
    parameters = crashReporter.getParameters()
    report.extraAdded = parameters.channel === 'stable' && parameters.build === '42'
    crashReporter.removeExtraParameter('build')
    parameters = crashReporter.getParameters()
    report.extraRemoved = parameters.channel === 'stable' && !Object.prototype.hasOwnProperty.call(parameters, 'build')
    crashReporter.setUploadToServer(true)
    report.uploadAfterSet = crashReporter.getUploadToServer()
    report.uploadedReportsEmpty = Array.isArray(crashReporter.getUploadedReports()) &&
      crashReporter.getUploadedReports().length === 0
    report.lastReportNull = crashReporter.getLastCrashReport() === null
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
