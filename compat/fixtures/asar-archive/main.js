const { app } = require('electron')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const ARCHIVE_BASE64 = [
  'BAAAABAEAAAMBAAABQQAAHsiZmlsZXMiOnsibWFpbi5qcyI6eyJzaXplIjoxOSwib2Zmc2V0IjoiMCIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiZjg3N2E4NjlkODI3YTdiOTg5Yjc4MzMwNTM4ZDEwNTRlOTk4MGM1MTlhMDE3NjU4ZjIzMzg3NjY1MzU1Zjg1MSIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbImY4NzdhODY5ZDgyN2E3Yjk4OWI3ODMzMDUzOGQxMDU0ZTk5ODBjNTE5YTAxNzY1OGYyMzM4NzY2NTM1NWY4NTEiXX19LCJuYXRpdmUiOnsiZmlsZXMiOnsiYWRkb24ubm9kZSI6eyJzaXplIjo2LCJvZmZzZXQiOiIxOSIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiYmVmMzJkMmMzMTVhMjg5NTc2ZjJhNjgyOGQyN2VkYjE2YmIzMTZhNGQ4NWMyNzFmMmQ3OTQwNDVmM2VhNjY4ZCIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbImJlZjMyZDJjMzE1YTI4OTU3NmYyYTY4MjhkMjdlZGIxNmJiMzE2YTRkODVjMjcxZjJkNzk0MDQ1ZjNlYTY2OGQiXX19fX0sInBhY2thZ2UuanNvbiI6eyJzaXplIjozNSwib2Zmc2V0IjoiMjUiLCJpbnRlZ3JpdHkiOnsiYWxnb3JpdGhtIjoiU0hBMjU2IiwiaGFzaCI6IjExZmRlMWNjYzczYzdmMTI3OTlhYmM2OTNhYTBhYjM2NjRlYjFiYmE3ZjgwZjgzODllMzdhNDNkZmYxMGNjNzIiLCJibG9ja1NpemUiOjQxOTQzMDQsImJsb2NrcyI6WyIxMWZkZTFjY2M3M2M3ZjEyNzk5YWJjNjkzYWEwYWIzNjY0ZWIxYmJhN2Y4MGY4Mzg5ZTM3YTQzZGZmMTBjYzcyIl19fSwicGtnIjp7ImZpbGVzIjp7ImluZGV4LmpzIjp7InNpemUiOjE5LCJvZmZzZXQiOiI2MCIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiNDdlZjRmNjYzM2M3NjEwMzgxNDE3MzIwNDM0ZTliZDdhNjVhYmYwODMyNjk0ZmJjN2Q4ODc5OTFiNDc4NzlhZSIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbIjQ3ZWY0ZjY2MzNjNzYxMDM4MTQxNzMyMDQzNGU5YmQ3YTY1YWJmMDgzMjY5NGZiYzdkODg3OTkxYjQ3ODc5YWUiXX19fX19fQAAAGNvbnNvbGUubG9nKCdhc2FyJyluYXRpdmV7Im5hbWUiOiJmaXh0dXJlIiwibWFpbiI6Im1haW4uanMifW1vZHVsZS5leHBvcnRzID0gNDI='
].join('')

app.whenReady().then(() => {
  const report = {
    readMain: false,
    requireIndex: false,
    unpackedRead: false,
    copySourceUnpacked: false,
    statSize: 0
  }

  try {
    const archivePath = process.env.ASAR_FIXTURE_ARCHIVE ||
      path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'electron-go-asar-')), 'app.asar')
    if (!process.env.ASAR_FIXTURE_ARCHIVE) {
      fs.writeFileSync(archivePath, Buffer.from(ARCHIVE_BASE64, 'base64'))
    }
    report.readMain = fs.readFileSync(path.join(archivePath, 'main.js'), 'utf8') === "console.log('asar')"
    report.requireIndex = fs.readFileSync(path.join(archivePath, 'pkg', 'index.js'), 'utf8') === 'module.exports = 42'
    report.unpackedRead = fs.readFileSync(path.join(archivePath, 'native', 'addon.node'), 'utf8') === 'native'
    report.copySourceUnpacked = false
    report.statSize = fs.statSync(path.join(archivePath, 'main.js')).size
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  }

  console.log(JSON.stringify(report))
  app.quit()
})
