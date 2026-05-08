const { app, BrowserWindow, ipcMain } = require('electron')
const path = require('path')
const config = require('./config.json')

ipcMain.handle('app:greeting', (_event, name) => `hello from preload IPC, ${name}`)

app.on('ready', () => {
  const win = new BrowserWindow({
    width: 900,
    height: 600,
    webPreferences: { preload: './preload.js' }
  })
  win.loadFile(path.join('renderer', config.entry))
  win.show()
  console.log('hello from an Electron-shaped app')
})
