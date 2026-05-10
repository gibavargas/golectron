const { app, clipboard } = require('electron')

app.whenReady().then(() => {
  const report = {
    htmlContainsPayload: false,
    rtfRoundTrip: false,
    rtfBytes: 0,
    bufferRoundTrip: false,
    bufferBytes: 0
  }

  try {
    const html = '<b>electron-go</b>'
    const rtf = '{\\\\rtf1\\\\ansi electron-go}'

    clipboard.writeHTML(html)
    report.htmlContainsPayload = clipboard.readHTML().includes(html)

    clipboard.writeRTF(rtf)
    const gotRTF = clipboard.readRTF()
    report.rtfBytes = gotRTF.length
    report.rtfRoundTrip = gotRTF === rtf

    const buffer = Buffer.from([4, 5, 6])
    clipboard.writeBuffer('electron-go/custom', buffer)
    const gotBuffer = clipboard.readBuffer('electron-go/custom')
    report.bufferBytes = gotBuffer.length
    report.bufferRoundTrip = Buffer.compare(gotBuffer, buffer) === 0
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
