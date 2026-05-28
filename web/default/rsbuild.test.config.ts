import path from 'path'
import { fileURLToPath } from 'url'
import { defineConfig } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [pluginReact()],
  source: {
    entry: {
      'enterprise-organization':
        './src/features/enterprise-organization/enterprise-organization.test.tsx',
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  output: {
    target: 'node',
    distPath: {
      root: '.tmp/tests',
    },
    filename: {
      js: '[name].mjs',
    },
    minify: false,
    cleanDistPath: true,
  },
})
