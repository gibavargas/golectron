const { contextBridge } = require('electron')

let duplicateRejected = false

const api = {
  version: '1.0.0',
  add: (a, b) => a + b,
  flags: [true],
  duplicateRejected: () => duplicateRejected
}

contextBridge.exposeInMainWorld('fixture', api)
try {
  contextBridge.exposeInMainWorld('fixture', { duplicate: true })
} catch (_error) {
  duplicateRejected = true
}
