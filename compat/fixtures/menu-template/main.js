const { app, Menu } = require('electron')

app.whenReady().then(() => {
  const report = {
    itemCount: 0,
    firstLabel: '',
    firstEnabled: false,
    checkboxLabel: '',
    checkboxChecked: false,
    submenuFound: false,
    submenuRole: ''
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
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
