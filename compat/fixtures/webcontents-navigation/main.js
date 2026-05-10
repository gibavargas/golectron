const { app, BrowserWindow } = require('electron')

app.whenReady().then(async () => {
  const report = {
    didStartNavigation: false,
    didNavigate: false,
    didFinishLoad: false,
    didNavigateInPage: false,
    canGoBack: false,
    canGoForward: false,
    loadingAfter: true,
    finalHash: '',
    historyLength: 0
  }

  try {
    const win = new BrowserWindow({ show: false })
    const contents = win.webContents
    contents.on('did-start-navigation', () => {
      report.didStartNavigation = true
    })
    contents.on('did-navigate', () => {
      report.didNavigate = true
    })
    contents.on('did-finish-load', () => {
      report.didFinishLoad = true
    })
    contents.on('did-navigate-in-page', () => {
      report.didNavigateInPage = true
    })

    const baseURL = 'data:text/html,<html><body>electron-go</body></html>'
    await contents.loadURL(baseURL)
    await contents.executeJavaScript("location.hash = 'section'")
    await new Promise((resolve) => setTimeout(resolve, 50))

    report.canGoBack = contents.navigationHistory.canGoBack()
    report.canGoForward = contents.navigationHistory.canGoForward()
    report.loadingAfter = contents.isLoading()
    report.finalHash = new URL(contents.getURL()).hash.slice(1)
    report.historyLength = contents.navigationHistory.getAllEntries().length
    win.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
