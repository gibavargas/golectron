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
    cookieOverwriteValue: false,
    cookieOverwriteChange: false,
    cookieRemoveResolved: false,
    cookieRemoved: false,
    cookieRemoveChange: false,
    partitionCookieIsolation: false,
    cacheClearResolved: false,
    storageClearResolved: false,
    storageClearRemovedCookies: false
  }

  try {
    const cookieChanges = []
    defaultSession.cookies.on('changed', (_event, cookie, cause, removed) => {
      if (cookie.name === 'sid') {
        cookieChanges.push({ value: cookie.value, cause, removed })
      }
    })
    await defaultSession.cookies.set({
      url: 'https://example.test/',
      name: 'sid',
      value: '1'
    })
    await defaultSession.cookies.set({
      url: 'https://example.test/',
      name: 'sid',
      value: '2'
    })
    const cookies = await defaultSession.cookies.get({
      url: 'https://example.test/',
      name: 'sid'
    })
    report.cookieCount = cookies.length
    report.cookieRoundTrip = cookies.some(cookie => cookie.name === 'sid' && cookie.value === '2')
    report.cookieOverwriteValue = cookies.length === 1 && cookies[0].value === '2'
    report.cookieOverwriteChange = cookieChanges.some(change => change.cause === 'overwrite' && change.removed)

    await defaultSession.cookies.remove('https://example.test/', 'sid')
    report.cookieRemoveResolved = true
    const afterRemove = await defaultSession.cookies.get({
      url: 'https://example.test/',
      name: 'sid'
    })
    report.cookieRemoved = afterRemove.length === 0
    report.cookieRemoveChange = cookieChanges.some(change => change.cause === 'explicit' && change.removed)

    await persistent.cookies.set({
      url: 'https://example.test/',
      name: 'sid',
      value: 'persist'
    })
    await inMemory.cookies.set({
      url: 'https://example.test/',
      name: 'sid',
      value: 'memory'
    })
    const persistentCookies = await persistent.cookies.get({ url: 'https://example.test/', name: 'sid' })
    const memoryCookies = await inMemory.cookies.get({ url: 'https://example.test/', name: 'sid' })
    report.partitionCookieIsolation = persistentCookies.some(cookie => cookie.value === 'persist') &&
      memoryCookies.some(cookie => cookie.value === 'memory') &&
      afterRemove.length === 0

    await defaultSession.clearCache()
    report.cacheClearResolved = true

    await defaultSession.cookies.set({
      url: 'https://example.test/',
      name: 'clearme',
      value: '1'
    })
    await defaultSession.clearStorageData({
      origins: ['https://example.test'],
      storages: ['cookies', 'localstorage']
    })
    report.storageClearResolved = true
    const afterStorageClear = await defaultSession.cookies.get({
      url: 'https://example.test/',
      name: 'clearme'
    })
    report.storageClearRemovedCookies = afterStorageClear.length === 0
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
