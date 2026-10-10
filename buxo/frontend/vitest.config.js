import { defineConfig } from 'vitest/config'

// Node 25 trae un localStorage global que, sen --localstorage-file, non
// funciona e tapa o de jsdom: desactívase só nos procesos dos tests.
// Os Node que non coñecen a opción (anteriores a 22) quedan como estaban.
const execArgv = process.allowedNodeEnvironmentFlags.has('--experimental-webstorage')
  ? ['--no-experimental-webstorage']
  : []

export default defineConfig({
  test: {
    poolOptions: {
      forks: { execArgv },
      threads: { execArgv },
    },
  },
})
