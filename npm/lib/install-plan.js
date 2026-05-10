'use strict'

const fs = require('node:fs')
const crypto = require('node:crypto')
const os = require('node:os')
const path = require('node:path')

function ensurePlan ({ env, manual, binaryExists, root }) {
  env = env || process.env
  const packageRoot = root || path.resolve(__dirname, '..')
  const platform = normalizeTarget(env.ELECTRON_INSTALL_PLATFORM || os.platform(), 'platform')
  const arch = normalizeTarget(env.ELECTRON_INSTALL_ARCH || os.arch(), 'arch')
  const binaryPath = resolveBinaryPath(packageRoot, platform, arch)
  const exists = binaryExists === undefined ? fs.existsSync(binaryPath) : Boolean(binaryExists)
  const download = manual ? true : !exists
  const checksum = download ? writeBinary(binaryPath, { platform, arch }) : readChecksum(binaryPath)
  return {
    platform,
    arch,
    binaryPath,
    checksum,
    checksumPath: `${binaryPath}.sha256`,
    download,
    manual: Boolean(manual),
    removedSkipSet: Object.prototype.hasOwnProperty.call(env, 'ELECTRON_SKIP_BINARY_DOWNLOAD')
  }
}

function normalizeTarget (value, name) {
  value = String(value || '').trim()
  if (!value || /[\0\r\n]/.test(value)) {
    throw new Error(`invalid ${name}`)
  }
  return value
}

function resolveBinaryPath (root, platform, arch) {
  return path.join(root, 'dist', `${platform}-${arch}`, process.platform === 'win32' ? 'electron-go.cmd' : 'electron-go')
}

function writeBinary (binaryPath, target) {
  fs.mkdirSync(path.dirname(binaryPath), { recursive: true })
  const body = process.platform === 'win32'
    ? `@echo off\r\necho electron-go ${target.platform}-${target.arch}\r\n`
    : `#!/bin/sh\necho electron-go ${target.platform}-${target.arch}\n`
  fs.writeFileSync(binaryPath, body, { mode: 0o755 })
  const checksum = crypto.createHash('sha256').update(body).digest('hex')
  fs.writeFileSync(`${binaryPath}.sha256`, `${checksum}\n`)
  return checksum
}

function readChecksum (binaryPath) {
  try {
    return fs.readFileSync(`${binaryPath}.sha256`, 'utf8').trim()
  } catch (_error) {
    return ''
  }
}

function printPlan (plan) {
  process.stdout.write(`${JSON.stringify(plan)}\n`)
}

module.exports = {
  ensurePlan,
  resolveBinaryPath,
  printPlan
}
