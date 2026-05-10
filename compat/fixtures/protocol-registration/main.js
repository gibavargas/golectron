const { app, net, protocol } = require('electron')

const report = {
  registeredPrivileged: false,
  allowExtensions: false,
  handledAfterRegister: false,
  fetchStatus: 0,
  fetchHeader: false,
  fetchBody: false,
  duplicateRejected: false,
  latePrivilegedRejected: false,
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
        status: 201,
        headers: {
          'content-type': 'text/plain',
          'x-eg-protocol': 'handled'
        }
      }))
      report.handledAfterRegister = Boolean(await protocol.isProtocolHandled('egtest'))
      const response = await net.fetch('egtest://fixture/path?mode=fetch')
      report.fetchStatus = response.status
      report.fetchHeader = response.headers.get('x-eg-protocol') === 'handled'
      report.fetchBody = await response.text() === 'ok'
      try {
        protocol.handle('egtest', () => new Response('duplicate'))
      } catch (_error) {
        report.duplicateRejected = true
      }
      try {
        protocol.registerSchemesAsPrivileged([{ scheme: 'late', privileges: { standard: true } }])
      } catch (_error) {
        report.latePrivilegedRejected = true
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
