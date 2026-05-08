const { app, BrowserWindow } = require('electron');

let win;

async function main() {
  await app.whenReady();

  win = new BrowserWindow({
    width: 800,
    height: 600,
    show: true,
    webPreferences: {
      contextIsolation: true,
      sandbox: true
    }
  });

  win.webContents.once('did-finish-load', () => {
    setImmediate(() => {
      win.close();
      app.quit();
    });
  });

  await win.loadFile('index.html');
}

app.on('window-all-closed', () => {
  app.quit();
});

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
  app.quit();
});
