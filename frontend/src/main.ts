import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import { applyStartupUIPreferences } from "@/stores/appSettings"
import "@vue-flow/core/dist/style.css"
import "@vue-flow/core/dist/theme-default.css"
import "./style.css"

applyStartupUIPreferences()

createApp(App).use(router).mount("#app")
