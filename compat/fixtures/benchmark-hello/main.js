const { app, BrowserWindow } = require('electron');

let win;
const traceStart = performance.now();
const trace = {};

function mark(name) {
  trace[name] = Math.round(performance.now() - traceStart);
}

function emitTrace() {
  console.log(`benchmark-trace: ${JSON.stringify(trace)}`);
}

async function main() {
  await app.whenReady();
  mark('app_ready');

  win = new BrowserWindow({
    width: 800,
    height: 600,
    show: true,
    webPreferences: {
      contextIsolation: true,
      sandbox: true
    }
  });
  mark('window_created');

  win.webContents.once('did-finish-load', () => {
    mark('did_finish_load');
    setImmediate(() => {
      mark('quit_requested');
      emitTrace();
      win.close();
      app.quit();
    });
  });

  mark('load_start');
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
