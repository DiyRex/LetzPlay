import path from "node:path"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// One remote bundle is shared by both servers. Vite builds to `web/dist`; the
// `scripts/build-web.sh` helper then copies it into the Android assets and the Go embed dir.
// Relative asset URLs (base: "./") keep it host-agnostic.
export default defineConfig({
  plugins: [react()],
  base: "./",
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  build: {
    outDir: path.resolve(__dirname, "./dist"),
    emptyOutDir: true,
  },
  server: {
    // During `npm run dev`, proxy API/WS to a running device/emulator on the LAN.
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
      "/ws": { target: "ws://localhost:8080", ws: true },
    },
  },
})
