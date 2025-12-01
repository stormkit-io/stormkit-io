import { defineConfig } from 'vite'
import dotenv from 'dotenv'
import sharedConfig from './vite.shared'

dotenv.config()

// https://vitejs.dev/config/
export default defineConfig({
  ...sharedConfig,
  build: {
    manifest: true,
    assetsInlineLimit: 0,
    rollupOptions: {
      input: 'index.html',
      // Material ui's "use client" directive causes a warning.
      // This function ignores those warnings.
      // See https://github.com/rollup/rollup/issues/4699#issuecomment-1571555307
      // for more information.
      onwarn(warning, warn) {
        if (
          warning.code === 'MODULE_LEVEL_DIRECTIVE' &&
          warning.message.includes('use client')
        ) {
          return
        }

        warn(warning)
      },
    },
    outDir: '.stormkit/public',
  },
})
