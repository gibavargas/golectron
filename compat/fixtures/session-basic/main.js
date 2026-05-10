const { app, session } = require('electron')

app.whenReady().then(async () => {
  const defaultSession = session.defaultSession
  const emptyPartition = session.fromPartition('')
  const persistent = session.fromPartition('persist:egtest')
  const inMemory = session.fromPartition('egtest-memory')

  const report = {
    defaultSameWithEmpty: defaultSession === emptyPartition,
    persistStoragePathNonempty: Boolean(persistent.storagePath),
    memoryStoragePathEmpty: !inMemory.storagePath,
    cookieRoundTrip: false,
    cookieCount: 0,
    cookieRemoveResolved: false,
    cookieRemoved: false,
    cacheClearResolved: false,
    storageClearResolved: false
  }

  try {
    await defaultSession.cookies.set({
      url: 'https://example.test/',
      name: 'sid',
      value: '1'
    })
    const cookies = await defaultSession.cookies.get({
      url: 'https://example.test/',
      name: 'sid'
    })
    report.cookieCount = cookies.length
    report.cookieRoundTrip = cookies.some(cookie => cookie.name === 'sid' && cookie.value === '1')

    await defaultSession.cookies.remove('https://example.test/', 'sid')
    report.cookieRemoveResolved = true
    const afterRemove = await defaultSession.cookies.get({
      url: 'https://example.test/',
      name: 'sid'
    })
    report.cookieRemoved = afterRemove.length === 0

    await defaultSession.clearCache()
    report.cacheClearResolved = true

    await defaultSession.clearStorageData({
      origins: ['https://example.test'],
      storages: ['cookies', 'localstorage']
    })
    report.storageClearResolved = true
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
