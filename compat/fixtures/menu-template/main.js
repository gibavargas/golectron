const { app, Menu, Tray, nativeImage } = require('electron')

app.whenReady().then(() => {
  const report = {
    itemCount: 0,
    firstLabel: '',
    firstEnabled: false,
    checkboxLabel: '',
    checkboxChecked: false,
    submenuFound: false,
    submenuRole: '',
    applicationMenuSet: false,
    applicationMenuRetrieved: false,
    trayCreated: false,
    trayTitleSet: false,
    trayToolTipSet: false,
    trayContextMenuSet: false,
    dynamicLabelUpdated: false,
    dynamicEnabledUpdated: false
  }

  try {
    const menu = Menu.buildFromTemplate([
      { id: 'open', label: ' Open ', accelerator: 'CmdOrCtrl+O' },
      { type: 'separator' },
      { id: 'toggle', label: 'Enabled', type: 'checkbox', checked: true, enabled: false },
      { id: 'view', label: 'View', submenu: [{ id: 'devtools', role: 'toggleDevTools' }] }
    ])
    const items = menu.items
    const submenuItem = items[3].submenu.getMenuItemById('devtools')

    report.itemCount = items.length
    report.firstLabel = items[0].label
    report.firstEnabled = items[0].enabled
    report.checkboxLabel = items[2].label
    report.checkboxChecked = items[2].checked
    report.submenuFound = Boolean(submenuItem)
    report.submenuRole = submenuItem ? submenuItem.role : ''
    items[0].label = 'Open File'
    items[0].enabled = false
    report.dynamicLabelUpdated = items[0].label === 'Open File'
    report.dynamicEnabledUpdated = items[0].enabled === false

    Menu.setApplicationMenu(menu)
    const applicationMenu = Menu.getApplicationMenu()
    report.applicationMenuSet = true
    report.applicationMenuRetrieved = Boolean(applicationMenu && applicationMenu.items.length === items.length)

    const trayImage = nativeImage.createFromDataURL('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lH9rqAAAAABJRU5ErkJggg==')
    const tray = new Tray(trayImage)
    report.trayCreated = !tray.isDestroyed()
    tray.setTitle('EG')
    report.trayTitleSet = true
    tray.setToolTip('Tooltip')
    report.trayToolTipSet = true
    tray.setContextMenu(menu)
    report.trayContextMenuSet = true
    tray.destroy()
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
