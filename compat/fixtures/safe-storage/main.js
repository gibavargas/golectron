const { app, safeStorage } = require('electron')

app.whenReady().then(() => {
  if (!safeStorage.isEncryptionAvailable() &&
      typeof safeStorage.setUsePlainTextEncryption === 'function') {
    safeStorage.setUsePlainTextEncryption(true)
  }

  const report = {
    available: safeStorage.isEncryptionAvailable(),
    asyncAvailable: typeof safeStorage.encryptStringAsync === 'function' &&
      typeof safeStorage.decryptStringAsync === 'function',
    backend: typeof safeStorage.getSelectedStorageBackend === 'function'
      ? safeStorage.getSelectedStorageBackend()
      : '',
    roundTrip: false,
    ciphertextBytes: 0
  }

  try {
    const ciphertext = safeStorage.encryptString('token')
    report.ciphertextBytes = ciphertext.length
    report.roundTrip = safeStorage.decryptString(ciphertext) === 'token'
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
