import { createRouter, createWebHashHistory } from "vue-router"
import Home from "@/pages/Home.vue"
import Devices from "@/pages/Devices.vue"
import LocalHub from "@/pages/LocalHub.vue"
import File from "@/pages/File.vue"
import Flow from "@/pages/Flow.vue"
import Debug from "@/pages/Debug.vue"
import Logs from "@/pages/Logs.vue"
import Presets from "@/pages/Presets.vue"
import TopicBus from "@/pages/TopicBus.vue"
import VarPool from "@/pages/VarPool.vue"
import FileTasks from "@/windows/FileTasks.vue"
import LogWindow from "@/windows/LogWindow.vue"

const routes = [
  { path: "/", redirect: "/home" },
  {
    path: "/home",
    name: "home",
    component: Home,
    meta: {
      title: "Home",
      subtitle: "Connect, authenticate, and monitor the current session."
    }
  },
  {
    path: "/devices",
    name: "devices",
    component: Devices,
    meta: {
      title: "Devices",
      subtitle: "Query nodes/devices from the management plane."
    }
  },
  {
    path: "/local-hub",
    name: "localHub",
    component: LocalHub,
    meta: {
      title: "Local Hub",
      subtitle: "Download and run hub_server as a sidecar process."
    }
  },
  {
    path: "/varpool",
    name: "varpool",
    component: VarPool,
    meta: {
      title: "VarPool",
      subtitle: "Inspect, set, and subscribe to variable pools."
    }
  },
  {
    path: "/topicbus",
    name: "topicbus",
    component: TopicBus,
    meta: {
      title: "TopicBus",
      subtitle: "Publish, subscribe, and replay topic events."
    }
  },
  {
    path: "/file",
    name: "file",
    component: File,
    meta: {
      title: "File Console",
      subtitle: "Browse remote nodes and manage transfer tasks."
    }
  },
  {
    path: "/file-tasks",
    name: "fileTasks",
    component: FileTasks,
    meta: {
      title: "File Tasks",
      layout: "window"
    }
  },
  {
    path: "/flow",
    name: "flow",
    component: Flow,
    meta: {
      title: "Flow",
      subtitle: "Build, deploy, and run flow graphs."
    }
  },
  {
    path: "/debug",
    name: "debug",
    component: Debug,
    meta: {
      title: "Debug",
      subtitle: "Craft headers, payloads, and send custom frames."
    }
  },
  {
    path: "/presets",
    name: "presets",
    component: Presets,
    meta: {
      title: "Presets",
      subtitle: "Run stress tests and reusable automation recipes."
    }
  },
  {
    path: "/logs",
    name: "logs",
    component: Logs,
    meta: {
      title: "Logs",
      subtitle: "Stream and filter session logs in real time."
    }
  },
  {
    path: "/log-window",
    name: "logWindow",
    component: LogWindow,
    meta: {
      title: "Log Window",
      layout: "window"
    }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
