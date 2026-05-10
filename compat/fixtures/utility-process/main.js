const { app, utilityProcess } = require('electron')
const path = require('node:path')

app.whenReady().then(() => {
  const report = {
    spawnEvent: false,
    messageEvent: false,
    exitEvent: false,
    exitCode: -1,
    stdioMode: 'pipe',
    stdoutPiped: false,
    stderrPiped: false,
    stdinPiped: false,
    stdinWrite: false,
    args: ['--fixture'],
    environmentValue: '',
    serviceName: 'fixture-service'
  }

  try {
    const child = utilityProcess.fork(
      path.join(__dirname, 'worker.js'),
      ['--fixture'],
      {
        env: { ...process.env, FIXTURE_ENV: 'ok' },
        stdio: 'pipe',
        serviceName: 'fixture-service',
        sandbox: true,
        disclaim: true
      }
    )

    report.stdoutPiped = Boolean(child.stdout)
    report.stderrPiped = Boolean(child.stderr)
    report.stdinPiped = Boolean(child.stdin)

    child.once('spawn', () => {
      report.spawnEvent = true
      if (child.stdin) {
        child.stdin.write('stdin')
        child.stdin.end()
        report.stdinWrite = true
      }
    })
    child.once('message', (message) => {
      report.messageEvent = Boolean(message && message.ready && message.arg && message.env === 'ok')
      report.environmentValue = message && message.env ? message.env : ''
    })
    child.once('exit', (code) => {
      report.exitEvent = true
      report.exitCode = code
      console.log(JSON.stringify(report))
      app.quit()
    })
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
    console.log(JSON.stringify(report))
    app.quit()
  }
})
