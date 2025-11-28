import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import { viteStaticCopy } from 'vite-plugin-static-copy'
import react from '@vitejs/plugin-react'
import dotenv from 'dotenv'

dotenv.config()
process.env.NODE_ENV = 'production'
const __dirname = path.dirname(fileURLToPath(import.meta.url))

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: [
      {
        find: /^~/,
        replacement: path.resolve(__dirname, 'src'),
      },
    ],
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json'],
  },
  build: {
    ssr: true,
    minify: false,
    manifest: true,
    rollupOptions: {
      preserveEntrySignatures: 'strict',
      input: { server: 'src/entry-server.tsx' },
      output: {
        dir: '.stormkit/server',
        format: 'esm',
        entryFileNames: '[name].mjs',
        preserveModules: true,
        preserveModulesRoot: 'src',
        exports: 'named',
      },
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
  },
  define: {
    ...Object.keys(process.env).reduce(
      (obj: Record<string, string>, key: string) => {
        obj[`process.env.${key}`] = JSON.stringify(process.env[key])
        return obj
      },
      {}
    ),
  },
  plugins: [
    react(),
    viteStaticCopy({
      targets: [
        {
          src: '.stormkit/public/index.html',
          dest: '../.stormkit/server',
        },
      ],
    }),
  ],
})
