const { app, BrowserWindow } = require('electron')

function rectJSON (bounds) {
  return {
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height
  }
}

function waitForEvent (target, event) {
  return new Promise((resolve) => target.once(event, resolve))
}

app.whenReady().then(async () => {
  const report = {
    initialVisible: false,
    afterShow: false,
    afterHide: true,
    eventOrder: [],
    bounds: { x: 0, y: 0, width: 0, height: 0 },
    normalBounds: { x: 0, y: 0, width: 0, height: 0 },
    enabledAfterDisable: true,
    enabledAfterEnable: false,
    minimizedAfterMinimize: false,
    minimizedAfterRestore: true,
    maximizedAfterMaximize: false,
    maximizedAfterUnmaximize: true,
    closePrevented: false,
    usableAfterPreventedClose: false,
    closeEvent: false,
    closedEvent: false,
    closed: false,
    enabledAfterEnd: true
  }

  try {
    const window = new BrowserWindow({ show: false, frame: false })
    report.initialVisible = window.isVisible()
    window.setBounds({ x: 10, y: 20, width: 500, height: 400 })
    window.setSize(320, 240)
    window.show()
    report.afterShow = window.isVisible()
    window.hide()
    report.afterHide = window.isVisible()

    window.setEnabled(false)
    report.enabledAfterDisable = window.isEnabled()
    window.setEnabled(true)
    report.enabledAfterEnable = window.isEnabled()

    window.show()
    report.eventOrder.push('show')
    const minimized = waitForEvent(window, 'minimize')
    window.minimize()
    await minimized
    report.eventOrder.push('minimize')
    report.minimizedAfterMinimize = window.isMinimized()
    const restored = waitForEvent(window, 'restore')
    window.restore()
    await restored
    report.eventOrder.push('restore')
    report.minimizedAfterRestore = window.isMinimized()
    const maximized = waitForEvent(window, 'maximize')
    window.maximize()
    await maximized
    report.eventOrder.push('maximize')
    report.maximizedAfterMaximize = window.isMaximized()
    const unmaximized = waitForEvent(window, 'unmaximize')
    window.unmaximize()
    await unmaximized
    report.eventOrder.push('unmaximize')
    report.maximizedAfterUnmaximize = window.isMaximized()

    let shouldPreventClose = true
    window.on('close', (event) => {
      report.closeEvent = true
      if (shouldPreventClose) {
        shouldPreventClose = false
        event.preventDefault()
        report.closePrevented = true
        report.eventOrder.push('close-prevented')
        return
      }
      report.eventOrder.push('close')
    })
    const closed = new Promise((resolve) => {
      window.on('closed', () => {
        report.eventOrder.push('closed')
        report.closedEvent = true
        report.closed = true
        report.enabledAfterEnd = false
        resolve()
      })
    })
    report.bounds = rectJSON(window.getBounds())
    report.normalBounds = rectJSON(window.getNormalBounds())
    window.close()
    report.usableAfterPreventedClose = !window.isDestroyed() && window.isEnabled()
    window.close()
    await closed
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
