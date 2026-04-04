import { existsSync, mkdirSync, writeFileSync } from "node:fs"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { spawnSync } from "node:child_process"

const FRONTEND_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const PLACEHOLDER_CONTENT = "This file keeps frontend/dist embeddable for go:embed before Wails builds real assets.\n"
const REQUIRED_DEPENDENCY_FILES = [
  "node_modules/vite/bin/vite.js",
  "node_modules/@vitejs/plugin-vue/package.json",
  "node_modules/@babel/parser/package.json"
]
const VALID_MODES = new Set(["dev", "build", "preview"])

const npmCommand = process.platform === "win32" ? "npm.cmd" : "npm"

export function listMissingDependencyFiles(rootDir, requiredFiles = REQUIRED_DEPENDENCY_FILES) {
  return requiredFiles.filter((relativePath) => !existsSync(join(rootDir, relativePath)))
}

export function restoreDistPlaceholder(rootDir, content = PLACEHOLDER_CONTENT) {
  const distDir = join(rootDir, "dist")
  mkdirSync(distDir, { recursive: true })
  writeFileSync(join(distDir, "placeholder.txt"), content)
}

export function ensureFrontendDependencies(rootDir, options = {}) {
  const {
    spawnSyncImpl = spawnSync,
    log = console.error
  } = options
  const missingBeforeInstall = listMissingDependencyFiles(rootDir)
  if (missingBeforeInstall.length === 0) {
    return false
  }

  log(
    `[run-vite] Missing frontend dependencies detected: ${missingBeforeInstall.join(", ")}. Running npm install...`
  )
  const result = spawnSyncImpl(npmCommand, ["install"], {
    cwd: rootDir,
    stdio: "inherit",
    env: {
      ...process.env,
      MYFLOWHUB_FRONTEND_SELF_HEAL: "1"
    }
  })
  if (typeof result.status === "number" && result.status !== 0) {
    const error = new Error(`[run-vite] npm install failed with exit code ${result.status}.`)
    error.exitCode = result.status
    throw error
  }
  if (result.error) {
    throw result.error
  }

  const missingAfterInstall = listMissingDependencyFiles(rootDir)
  if (missingAfterInstall.length > 0) {
    throw new Error(
      `[run-vite] Frontend dependencies are still incomplete after npm install: ${missingAfterInstall.join(", ")}`
    )
  }
  return true
}

export function runVite(mode, rootDir, options = {}) {
  if (!VALID_MODES.has(mode)) {
    throw new Error(`[run-vite] Unsupported mode '${mode}'. Expected one of: ${Array.from(VALID_MODES).join(", ")}.`)
  }
  const { spawnSyncImpl = spawnSync } = options
  const viteEntrypoint = join(rootDir, "node_modules", "vite", "bin", "vite.js")
  const args = [viteEntrypoint]
  if (mode !== "dev") {
    args.push(mode)
  }
  const result = spawnSyncImpl(process.execPath, args, {
    cwd: rootDir,
    stdio: "inherit",
    env: process.env
  })
  if (result.error) {
    throw result.error
  }
  return result.status ?? 1
}

export async function main(argv = process.argv.slice(2), options = {}) {
  const [mode = "dev"] = argv
  ensureFrontendDependencies(FRONTEND_ROOT, options)
  const exitCode = runVite(mode, FRONTEND_ROOT, options)
  if (exitCode === 0 && mode === "build") {
    restoreDistPlaceholder(FRONTEND_ROOT)
  }
  return exitCode
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  main()
    .then((exitCode) => {
      process.exit(exitCode)
    })
    .catch((error) => {
      console.error(error instanceof Error ? error.message : error)
      process.exit(typeof error?.exitCode === "number" ? error.exitCode : 1)
    })
}
