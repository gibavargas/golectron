const { app, contentTracing } = require('electron')
const os = require('node:os')
const path = require('node:path')

app.whenReady().then(async () => {
  const report = {
    started: false,
    stopped: false,
    requestedPathReturned: false,
    categories: ['electron'],
    heapProfiling: false
  }

  try {
    const tracePath = path.join(os.tmpdir(), 'electron-go-trace.json')
    await contentTracing.startRecording({ included_categories: ['electron'] })
    report.started = true
    const resultPath = await contentTracing.stopRecording(tracePath)
    report.stopped = true
    report.requestedPathReturned = resultPath === tracePath
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
