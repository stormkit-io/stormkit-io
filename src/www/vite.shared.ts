import { fileURLToPath } from 'node:url'
import path from 'node:path'
import react from '@vitejs/plugin-react'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default {
  resolve: {
    alias: [
      {
        find: /^~/,
        replacement: path.resolve(__dirname, 'src'),
      },
      {
        find: '@docs',
        replacement: path.resolve(__dirname, '../../docs'),
      },
      {
        find: '@content',
        replacement: path.resolve(__dirname, '../../content'),
      },
    ],
    extensions: ['.mjs', '.cjs', '.js', '.ts', '.jsx', '.tsx', '.json'],
  },
  plugins: [react({ jsxImportSource: '@emotion/react' })],
}
