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
      'enterprise-dingtalk':
        './src/features/enterprise-dingtalk/enterprise-dingtalk.test.tsx',
      'enterprise-alerts':
        './src/features/enterprise-alerts/enterprise-alerts.test.tsx',
      'enterprise-usage':
        './src/features/enterprise-usage/enterprise-usage.test.tsx',
      'subscription-plans-card':
        './src/features/wallet/components/subscription-plans-card.test.tsx',
      'user-subscriptions-dialog':
        './src/features/subscriptions/components/dialogs/user-subscriptions-dialog.test.tsx',
      'quota-settings-section':
        './src/features/system-settings/general/quota-settings-section.test.tsx',
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
