// 本文件启动 Vue 前端，并装配路由、i18n 和共享壳层。

import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import { applyStartupUIPreferences } from "@/stores/appSettings"
import { applyStartupLocale } from "@/stores/language"
import "@vue-flow/core/dist/style.css"
import "@vue-flow/core/dist/theme-default.css"
import "./style.css"

applyStartupLocale()
applyStartupUIPreferences()

createApp(App).use(router).mount("#app")
