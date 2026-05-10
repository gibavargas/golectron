const { app, net } = require('electron')
const http = require('node:http')

function listen (server) {
  return new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => resolve(server.address().port))
  })
}

app.whenReady().then(async () => {
  const report = {
    method: '',
    path: '',
    queryMode: '',
    header: '',
    body: '',
    redirectPolicy: 'manual',
    ended: false
  }

  const server = http.createServer((req, res) => {
    const chunks = []
    req.on('data', (chunk) => chunks.push(chunk))
    req.on('end', () => {
      const requestUrl = new URL(req.url, 'http://127.0.0.1')
      report.method = req.method
      report.path = requestUrl.pathname
      report.queryMode = requestUrl.searchParams.get('mode') || ''
      report.header = req.headers['x-test'] || ''
      report.body = Buffer.concat(chunks).toString()
      res.statusCode = 204
      res.end()
    })
  })

  try {
    const port = await listen(server)
    await new Promise((resolve, reject) => {
      const request = net.request({
        method: 'POST',
        url: `http://127.0.0.1:${port}/request?mode=manual`,
        redirect: 'manual'
      })
      request.setHeader('X-Test', 'one')
      request.on('response', (response) => {
        response.on('end', resolve)
        response.on('error', reject)
        response.resume()
      })
      request.on('error', reject)
      request.write('payload')
      request.end()
      report.ended = true
    })
  } catch (error) {
    report.error = error && error.message ? error.message : String(error)
  } finally {
    server.close()
  }

  console.log(JSON.stringify(report))
  app.quit()
})
