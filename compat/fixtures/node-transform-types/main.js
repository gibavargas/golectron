const { app } = require('electron')

app.whenReady().then(() => {
  const args = [...process.execArgv, ...process.argv]
  const supported = Boolean(
    process.allowedNodeEnvironmentFlags &&
      process.allowedNodeEnvironmentFlags.has('--experimental-transform-types')
  )

  const report = {
    supported,
    experimentalTransformTypes: supported || args.includes('--experimental-transform-types'),
    execArgvIncludesFlag: process.execArgv.includes('--experimental-transform-types'),
    node: process.versions.node
  }

  console.log(JSON.stringify(report))
  app.quit()
})
