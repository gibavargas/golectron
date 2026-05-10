#!/usr/bin/env node
'use strict'

const { ensurePlan, printPlan } = require('./lib/install-plan')

try {
  const manual = process.argv.includes('--manual')
  const plan = ensurePlan({
    env: process.env,
    manual,
    binaryExists: !manual,
    root: process.env.ELECTRON_GO_NPM_ROOT
  })
  printPlan(plan)
} catch (error) {
  console.error(error && error.message ? error.message : String(error))
  process.exit(1)
}
