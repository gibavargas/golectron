const { ipcRenderer } = require('electron')

ipcRenderer.invoke('app:greeting', 'Golectron').then(message => {
  console.log(message)
})
