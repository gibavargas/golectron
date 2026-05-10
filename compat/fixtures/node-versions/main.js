const { app } = require('electron')
const path = require('node:path')

app.whenReady().then(async () => {
  const url = await import('node:url')
  const report = {
    electron: process.versions.electron,
    chrome: process.versions.chrome,
    node: process.versions.node,
    v8: process.versions.v8,
    modules: process.versions.modules,
    cjsRequireWorks: path.basename('/tmp/fixture.txt') === 'fixture.txt',
    esmImportWorks: typeof url.pathToFileURL === 'function',
    nativeABIReported: Boolean(process.versions.modules)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
