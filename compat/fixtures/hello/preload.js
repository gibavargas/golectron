const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('fixture', {
  ping: async () => {
    const result = await ipcRenderer.invoke('fixture:ping');
    ipcRenderer.send('fixture:done', result);
    return result;
  }
});
