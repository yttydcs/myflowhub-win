// Context: defines the Vitest environment and test setup for the Win frontend.

import { defineConfig } from "vitest/config"
import vue from "@vitejs/plugin-vue"
import { fileURLToPath, URL } from "url"

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "../../wailsjs/go/main/App": fileURLToPath(new URL("./src/test/wails_main_app.stub.ts", import.meta.url))
    }
  },
  test: {
    environment: "node"
  }
})
