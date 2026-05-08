const { app, BrowserWindow } = require('electron')
const path = require('path')
const config = require('./config.json')

app.on('ready', () => {
  const win = new BrowserWindow({ width: 900, height: 600 })
  win.loadFile(path.join('renderer', config.entry))
  win.show()
  console.log('hello from an Electron-shaped app')
})
