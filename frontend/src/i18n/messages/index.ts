// Context: aggregates the domain-specific message bundles used by the Win frontend.

import type { AppLocale, LocaleMessages } from "../types"
import { automationZhCN } from "./automation"
import { commonZhCN } from "./common"
import { fileZhCN } from "./file"
import { operationsZhCN } from "./operations"
import { sessionZhCN } from "./session"
import { settingsZhCN } from "./settings"
import { shellZhCN } from "./shell"
import { showcaseZhCN } from "./showcase"
import { signalsZhCN } from "./signals"
import { storesZhCN } from "./stores"

const zhCN: LocaleMessages = {
  ...commonZhCN,
  ...shellZhCN,
  ...settingsZhCN,
  ...sessionZhCN,
  ...operationsZhCN,
  ...signalsZhCN,
  ...automationZhCN,
  ...fileZhCN,
  ...showcaseZhCN,
  ...storesZhCN
}

const en: LocaleMessages = {}

export const messages: Record<AppLocale, LocaleMessages> = {
  en,
  "zh-CN": zhCN
}
