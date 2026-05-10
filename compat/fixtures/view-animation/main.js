const { app, View } = require('electron')

function rectJSON (bounds) {
  return {
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height
  }
}

app.whenReady().then(async () => {
  const report = {
    animated: true,
    durationMS: 300,
    easing: 'ease-in-out',
    boundsChanged: false,
    backgroundBlurMethodAvailable: false,
    backgroundBlurAccepted: false,
    bounds: { x: 0, y: 0, width: 0, height: 0 }
  }

  try {
    const view = new View()
    view.on('bounds-changed', () => {
      report.boundsChanged = true
    })
    view.setBounds({ x: 1, y: 2, width: 300, height: 200 }, {
      animate: {
        duration: 300,
        easing: 'ease-in-out'
      }
    })
    report.backgroundBlurMethodAvailable = typeof view.setBackgroundBlur === 'function'
    if (report.backgroundBlurMethodAvailable) {
      view.setBackgroundBlur({ type: 'blur', radius: 20 })
      report.backgroundBlurAccepted = true
    }
    await new Promise((resolve) => setTimeout(resolve, 350))
    report.bounds = rectJSON(view.getBounds())
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
