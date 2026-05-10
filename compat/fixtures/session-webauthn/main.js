const { app, session } = require('electron')

app.whenReady().then(() => {
  const report = {
    handlerCalled: false,
    requestId: '',
    origin: '',
    credentialCount: 0,
    selectedCredential: ''
  }

  try {
    session.defaultSession.once('select-webauthn-account', (_event, details, callback) => {
      report.handlerCalled = true
      report.requestId = String(details.requestId || '').trim()
      report.origin = String(details.origin || '').trim()
      report.credentialCount = Array.isArray(details.credentials) ? details.credentials.length : 0
      callback('credential-2')
    })
    session.defaultSession.emit(
      'select-webauthn-account',
      { preventDefault () {} },
      {
        requestId: ' request ',
        origin: ' https://example.test ',
        credentials: [
          { credentialId: 'credential-1', rpId: 'example.test', userName: 'one' },
          { credentialId: 'credential-2', rpId: 'example.test', userName: 'two' }
        ]
      },
      (selected) => {
        report.selectedCredential = selected
      }
    )
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
