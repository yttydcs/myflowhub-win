// 本文件定义 Win 前端的 Vitest 运行环境与测试装配。

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
