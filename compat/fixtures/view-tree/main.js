const { app, BaseWindow, WebContentsView } = require('electron')

function rectJSON (bounds) {
  return {
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height
  }
}

function childWidths (view) {
  return view.children.map((child) => child.getBounds().width)
}

app.whenReady().then(async () => {
  const report = {
    boundsChanged: false,
    rootChildWidths: [],
    afterRemoveWidths: [],
    afterReparentWidths: [],
    newParentChildWidths: [],
    liveWebContentsLoaded: false,
    liveWebContentsURL: '',
    liveWebContentsParented: false,
    liveWebContentsDetached: false,
    probeBounds: { x: 0, y: 0, width: 0, height: 0 }
  }

  try {
    const rootWindow = new BaseWindow({ show: false, width: 800, height: 600 })
    const secondWindow = new BaseWindow({ show: false, width: 800, height: 600 })
    const a = new WebContentsView()
    const b = new WebContentsView()
    const c = new WebContentsView()
    const live = new WebContentsView()

    c.on('bounds-changed', () => {
      report.boundsChanged = true
    })
    a.setBounds({ x: 0, y: 0, width: 100, height: 20 })
    b.setBounds({ x: 0, y: 0, width: 200, height: 20 })
    c.setBounds({ x: 7, y: 8, width: 300, height: 200 })

    rootWindow.contentView.addChildView(a)
    rootWindow.contentView.addChildView(b)
    rootWindow.contentView.addChildView(c, 1)
    report.rootChildWidths = childWidths(rootWindow.contentView)

    rootWindow.contentView.removeChildView(b)
    report.afterRemoveWidths = childWidths(rootWindow.contentView)

    secondWindow.contentView.addChildView(a)
    report.afterReparentWidths = childWidths(rootWindow.contentView)
    report.newParentChildWidths = childWidths(secondWindow.contentView)
    report.probeBounds = rectJSON(c.getBounds())

    live.setBounds({ x: 0, y: 0, width: 400, height: 300 })
    secondWindow.contentView.addChildView(live)
    report.liveWebContentsParented = secondWindow.contentView.children.includes(live)
    await live.webContents.loadURL('data:text/html,<html><body>webcontentsview</body></html>')
    report.liveWebContentsLoaded = live.webContents.getURL().startsWith('data:text/html')
    report.liveWebContentsURL = new URL(live.webContents.getURL()).protocol
    secondWindow.contentView.removeChildView(live)
    report.liveWebContentsDetached = !secondWindow.contentView.children.includes(live)

    rootWindow.destroy()
    secondWindow.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
