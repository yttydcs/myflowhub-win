import { createRouter, createWebHashHistory } from "vue-router"
import Home from "@/pages/Home.vue"
import Devices from "@/pages/Devices.vue"
import LocalHub from "@/pages/LocalHub.vue"
import File from "@/pages/File.vue"
import Flow from "@/pages/Flow.vue"
import Debug from "@/pages/Debug.vue"
import Logs from "@/pages/Logs.vue"
import Presets from "@/pages/Presets.vue"
import Settings from "@/pages/Settings.vue"
import Stream from "@/pages/Stream.vue"
import TopicBus from "@/pages/TopicBus.vue"
import VarPool from "@/pages/VarPool.vue"
import AccessPolicy from "@/pages/AccessPolicy.vue"
import RegistrationApprovals from "@/pages/RegistrationApprovals.vue"
import PermitIssuance from "@/pages/PermitIssuance.vue"
import ShowcaseCenter from "@/pages/ShowcaseCenter.vue"
import { readStartupRoutePath } from "@/stores/appSettings"
import FileTasks from "@/windows/FileTasks.vue"
import FlowEditorWindow from "@/windows/FlowEditorWindow.vue"
import LogWindow from "@/windows/LogWindow.vue"
import ShowcaseEditorWindow from "@/windows/ShowcaseEditorWindow.vue"
import ShowcaseWindow from "@/windows/ShowcaseWindow.vue"
import TopicBusWindow from "@/windows/TopicBusWindow.vue"

const routes = [
  { path: "/", redirect: () => readStartupRoutePath() },
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
    path: "/stream",
    name: "stream",
    component: Stream,
    meta: {
      title: "Stream",
      subtitle: "Query typed sources and consumers, connect deliveries, and inspect runtime traffic."
    }
  },
  {
    path: "/topicbus-window",
    name: "topicbusWindow",
    component: TopicBusWindow,
    meta: {
      title: "TopicBus Window",
      layout: "window",
      windowMode: "full-bleed"
    }
  },
  {
    path: "/showcase",
    name: "showcase",
    component: ShowcaseCenter,
    meta: {
      title: "Showcase Center",
      subtitle: "Manage showcase screens and open dedicated editor or viewer windows."
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
      subtitle: "Manage local projects, deployments, and editor windows."
    }
  },
  {
    path: "/flow-editor-window",
    name: "flowEditorWindow",
    component: FlowEditorWindow,
    meta: {
      title: "Flow Editor",
      layout: "window",
      windowMode: "full-bleed"
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
    path: "/access-policy",
    name: "accessPolicy",
    component: AccessPolicy,
    meta: {
      title: "Access Policy",
      subtitle: "Manage authority access rules, role catalog, and runtime policy validation."
    }
  },
  {
    path: "/registration-approvals",
    name: "registrationApprovals",
    component: RegistrationApprovals,
    meta: {
      title: "Registration Approvals",
      subtitle: "Review and decide pending first-register requests."
    }
  },
  {
    path: "/permit-issuance",
    name: "permitIssuance",
    component: PermitIssuance,
    meta: {
      title: "Permit Issuance",
      subtitle: "Issue and revoke join permits for expected devices."
    }
  },
  {
    path: "/permissions",
    redirect: "/access-policy"
  },
  {
    path: "/approvals",
    redirect: "/registration-approvals"
  },
  {
    path: "/permits",
    redirect: "/permit-issuance"
  },
  {
    path: "/settings",
    name: "settings",
    component: Settings,
    meta: {
      title: "Settings",
      subtitle: "Manage app defaults, UI preferences, and version information."
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
  },
  {
    path: "/showcase-window",
    name: "showcaseWindow",
    component: ShowcaseWindow,
    meta: {
      title: "Showcase Viewer",
      layout: "window"
    }
  },
  {
    path: "/showcase-editor-window",
    name: "showcaseEditorWindow",
    component: ShowcaseEditorWindow,
    meta: {
      title: "Showcase Editor",
      layout: "window",
      windowMode: "full-bleed"
    }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
