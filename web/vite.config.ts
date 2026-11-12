import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    // W dev frontend chodzi na 5173, a Go na 8080. Proxy sprawia, że przeglądarka
    // widzi jeden origin — dzięki temu nie ma CORS-a i kod fetchujący jest
    // identyczny w dev i w prod (zawsze względne "/api/...").
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: false,
      },
    },
  },
  build: {
    // Ten katalog trafia do binarki Go przez go:embed (patrz web/embed.go).
    outDir: 'dist',
    emptyOutDir: true,
    // Domyślny próg 500 kB krzyczy przy każdym buildzie na paczkę three.js.
    // Ona JEST duża i to jest w porządku: ładuje się leniwie, tylko na stronie
    // rzeczy z modelem 3D. Ostrzeżenie, które zawsze znaczy "wszystko gra",
    // uczy ignorowania ostrzeżeń.
    chunkSizeWarningLimit: 700,
  },
})
