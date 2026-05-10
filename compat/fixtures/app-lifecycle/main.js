const { app } = require('electron')

const report = {
  ready: false,
  whenReady: false,
  eventOrder: [],
  cancelledBeforeQuit: false,
  quitExitCode: 0
}

let shouldCancelBeforeQuit = true

app.on('ready', () => {
  report.eventOrder.push('ready')
})

app.on('before-quit', (event) => {
  if (shouldCancelBeforeQuit) {
    shouldCancelBeforeQuit = false
    event.preventDefault()
    report.cancelledBeforeQuit = true
    report.eventOrder.push('before-quit-cancelled')
    setImmediate(() => app.quit())
    return
  }
  report.eventOrder.push('before-quit')
  report.quitExitCode = event.exitCode || 0
})

app.on('will-quit', (event) => {
  report.eventOrder.push('will-quit')
  report.quitExitCode = event.exitCode || 0
})

app.on('quit', (_event, exitCode) => {
  report.eventOrder.push('quit')
  report.quitExitCode = exitCode || 0
  console.log(JSON.stringify(report))
})

app.whenReady().then(() => {
  report.whenReady = true
  report.eventOrder.unshift('when-ready')
  report.ready = app.isReady()
  app.quit()
})
