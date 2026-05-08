const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('fixture', {
  ping: () => ipcRenderer.invoke('fixture:ping')
});
