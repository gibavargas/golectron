#!/usr/bin/env node
'use strict'

const { spawnSync } = require('node:child_process')
const { ensurePlan, printPlan } = require('../lib/install-plan')

try {
  const plan = ensurePlan({
    env: process.env,
    manual: false,
    root: process.env.ELECTRON_GO_NPM_ROOT
  })
  if (process.env.ELECTRON_GO_NPM_PRINT_PLAN === '1') {
    printPlan(plan)
    process.exit(0)
  }
  const result = spawnSync(plan.binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    env: process.env
  })
  if (result.error) {
    throw result.error
  }
  process.exit(result.status || 0)
} catch (error) {
  console.error(error && error.message ? error.message : String(error))
  process.exit(1)
}
