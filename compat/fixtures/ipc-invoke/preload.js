const { ipcRenderer } = require('electron')

async function rejected (promise) {
  try {
    await promise
    return false
  } catch (_error) {
    return true
  }
}

window.addEventListener('DOMContentLoaded', async () => {
  const channel = new MessageChannel()
  ipcRenderer.postMessage('fixture:renderer-port', null, [channel.port1])
  channel.port2.postMessage('from-renderer-port')
  const report = {
    invokePong: await ipcRenderer.invoke('fixture:ping', 'renderer') === 'pong',
    onceFirst: await ipcRenderer.invoke('fixture:once') === 1,
    onceSecondRejected: await rejected(ipcRenderer.invoke('fixture:once')),
    removedRejected: await rejected(ipcRenderer.invoke('fixture:gone')),
    missingRejected: await rejected(ipcRenderer.invoke('fixture:missing'))
  }
  ipcRenderer.send('fixture:done', report)
})
