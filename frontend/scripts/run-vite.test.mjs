import { mkdtempSync, mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { tmpdir } from "node:os"
import { afterEach, describe, expect, it, vi } from "vitest"

import {
  ensureFrontendDependencies,
  listMissingDependencyFiles,
  restoreDistPlaceholder
} from "./run-vite.mjs"

function makeFrontendRoot() {
  return mkdtempSync(join(tmpdir(), "myflowhub-run-vite-"))
}

function touch(relativePath, rootDir) {
  const fullPath = join(rootDir, relativePath)
  mkdirSync(dirname(fullPath), { recursive: true })
  writeFileSync(fullPath, "")
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe("run-vite guard", () => {
  it("detects missing dependency files", () => {
    const rootDir = makeFrontendRoot()

    expect(listMissingDependencyFiles(rootDir)).toEqual([
      "node_modules/vite/bin/vite.js",
      "node_modules/@vitejs/plugin-vue/package.json",
      "node_modules/@babel/parser/package.json"
    ])
  })

  it("skips npm install when required files already exist", () => {
    const rootDir = makeFrontendRoot()
    touch("node_modules/vite/bin/vite.js", rootDir)
    touch("node_modules/@vitejs/plugin-vue/package.json", rootDir)
    touch("node_modules/@babel/parser/package.json", rootDir)
    const spawnSyncImpl = vi.fn()

    const installed = ensureFrontendDependencies(rootDir, { spawnSyncImpl, log: vi.fn() })

    expect(installed).toBe(false)
    expect(spawnSyncImpl).not.toHaveBeenCalled()
  })

  it("runs npm install once when dependencies are missing", () => {
    const rootDir = makeFrontendRoot()
    const spawnSyncImpl = vi.fn(() => {
      touch("node_modules/vite/bin/vite.js", rootDir)
      touch("node_modules/@vitejs/plugin-vue/package.json", rootDir)
      touch("node_modules/@babel/parser/package.json", rootDir)
      return { status: 0 }
    })

    const installed = ensureFrontendDependencies(rootDir, { spawnSyncImpl, log: vi.fn() })

    expect(installed).toBe(true)
    expect(spawnSyncImpl).toHaveBeenCalledTimes(1)
    expect(listMissingDependencyFiles(rootDir)).toEqual([])
  })

  it("fails explicitly when npm install does not restore required files", () => {
    const rootDir = makeFrontendRoot()
    const spawnSyncImpl = vi.fn(() => ({ status: 0 }))

    expect(() => ensureFrontendDependencies(rootDir, { spawnSyncImpl, log: vi.fn() })).toThrow(
      "Frontend dependencies are still incomplete after npm install"
    )
  })

  it("restores the embed placeholder file", () => {
    const rootDir = makeFrontendRoot()

    restoreDistPlaceholder(rootDir)

    expect(readFileSync(join(rootDir, "dist", "placeholder.txt"), "utf8")).toBe(
      "This file keeps frontend/dist embeddable for go:embed before Wails builds real assets.\n"
    )
  })
})
