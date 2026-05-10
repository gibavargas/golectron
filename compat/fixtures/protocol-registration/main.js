const { app, protocol } = require('electron')

const report = {
  registeredPrivileged: false,
  allowExtensions: false,
  handledAfterRegister: false,
  duplicateRejected: false,
  handledAfterRemove: false
}

try {
  protocol.registerSchemesAsPrivileged([
    {
      scheme: 'egtest',
      privileges: {
        standard: true,
        secure: true,
        supportFetchAPI: true,
        allowExtensions: true
      }
    }
  ])
  report.registeredPrivileged = true
  report.allowExtensions = true
} catch (error) {
  report.error = error && error.message ? error.message : String(error)
}

app.whenReady().then(async () => {
  if (!report.error) {
    try {
      protocol.handle('egtest', () => new Response('ok', {
        status: 200,
        headers: { 'content-type': 'text/plain' }
      }))
      report.handledAfterRegister = Boolean(await protocol.isProtocolHandled('egtest'))
      try {
        protocol.handle('egtest', () => new Response('duplicate'))
      } catch (_error) {
        report.duplicateRejected = true
      }
      protocol.unhandle('egtest')
      report.handledAfterRemove = Boolean(await protocol.isProtocolHandled('egtest'))
    } catch (error) {
      report.error = error && error.message ? error.message : String(error)
    }
  }

  console.log(JSON.stringify(report))
  app.quit()
})
