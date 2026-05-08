const { app, BrowserWindow, ipcMain } = require('electron');

async function main() {
  await app.whenReady();

  const win = new BrowserWindow({
    width: 800,
    height: 600,
    webPreferences: {
      preload: `${__dirname}/preload.js`,
      contextIsolation: true,
      sandbox: true
    }
  });

  ipcMain.handle('fixture:ping', () => 'pong');
  await win.loadFile('index.html');
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
