const { app, BrowserWindow } = require('electron')

app.on('ready', () => {
  const win = new BrowserWindow({ width: 900, height: 600 })
  win.loadURL('https://example.com')
  win.show()
  console.log('hello from an Electron-shaped app')
})
