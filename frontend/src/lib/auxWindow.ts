import { Environment } from "../../wailsjs/runtime/runtime"

export type AuxWindowResult = "opened" | "navigated" | "blocked"

type OpenAuxWindowOptions = {
  routePath: string
  name: string
  size: string
}

let packagedRuntimePromise: Promise<boolean> | null = null

const normalizeRoutePath = (routePath: string): string => {
  const trimmed = String(routePath ?? "").trim()
  if (!trimmed.startsWith("#/")) {
    throw new Error("Aux window route must start with '#/'.")
  }
  return trimmed
}

const detectPackagedRuntime = async (): Promise<boolean> => {
  if (typeof window === "undefined" || typeof (window as any)?.runtime === "undefined") {
    return false
  }
  if (!packagedRuntimePromise) {
    packagedRuntimePromise = Environment()
      .then((info) => String(info?.buildType ?? "").trim().toLowerCase() !== "dev")
      .catch(() => false)
  }
  return packagedRuntimePromise
}

const navigateCurrentWindow = (routePath: string) => {
  window.location.hash = routePath
}

export const openAuxWindow = async ({
  routePath,
  name,
  size
}: OpenAuxWindowOptions): Promise<AuxWindowResult> => {
  const normalizedRoute = normalizeRoutePath(routePath)
  if (await detectPackagedRuntime()) {
    navigateCurrentWindow(normalizedRoute)
    return "navigated"
  }

  const base = window.location.href.split("#")[0]
  const win = window.open(`${base}${normalizedRoute}`, name, size)
  if (win) {
    win.focus()
    return "opened"
  }
  return "blocked"
}

export const resetAuxWindowRuntimeCacheForTests = () => {
  packagedRuntimePromise = null
}
