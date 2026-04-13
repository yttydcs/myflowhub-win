// Context: boots the Vue frontend and installs the shared router, i18n, and shell-level providers.

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
