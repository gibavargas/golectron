let stdin = ''

process.stdin.on('data', (chunk) => {
  stdin += chunk.toString()
})

process.stdin.on('end', () => {
  process.parentPort.postMessage({
    ready: true,
    arg: process.argv.includes('--fixture'),
    env: process.env.FIXTURE_ENV,
    stdin
  })

  process.exit(0)
})
