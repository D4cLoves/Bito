import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig, type Plugin } from 'vite'

function phosphorWeights(drop: string[]): Plugin {
  const icon = /@phosphor-icons[\\/]vue[\\/]dist[\\/]icons[\\/]Ph\w+\.vue\.mjs$/
  const branch = new RegExp(`[\\w$]+\\.value === "(?:${drop.join('|')})" \\? \\([\\w$]+\\(\\), [\\w$]+\\("g", [\\w$]+, [\\w$]+\\)\\) : `, 'g')
  return {
    name: 'phosphor-weights',
    apply: 'build',
    transform(code, id) {
      if (!icon.test(id)) return null
      return { code: code.replace(branch, ''), map: null }
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    phosphorWeights(['light', 'thin']),
  ],
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://127.0.0.1:8080',
        ws: true,
      },
    },
  },
})
