const { app, clipboard } = require('electron')

app.whenReady().then(() => {
  const report = {
    textRoundTrip: false,
    text: ''
  }

  try {
    clipboard.writeText('electron-go clipboard')
    report.text = clipboard.readText()
    report.textRoundTrip = report.text === 'electron-go clipboard'
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
