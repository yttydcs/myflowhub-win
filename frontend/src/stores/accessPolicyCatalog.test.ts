// 本文件覆盖 `accessPolicyCatalog` store 的行为。

import { describe, expect, it } from "vitest"
import {
  accessPolicyPermissionOptions,
  findPermissionCatalogItem,
  mergeSelectedAndUnknownPerms,
  rolePresetPerms,
  splitKnownAndUnknownPerms
} from "./accessPolicyCatalog"

describe("accessPolicyCatalog", () => {
  it("splits known and unknown permissions while collapsing wildcard selections", () => {
    const result = splitKnownAndUnknownPerms([
      "custom.scope",
      "file.write",
      "*",
      "auth.revoke"
    ])

    expect(result.knownPerms).toEqual(["*"])
    expect(result.unknownPerms).toEqual(["custom.scope"])
  })

  it("merges known and unknown permissions without duplicating known items", () => {
    const merged = mergeSelectedAndUnknownPerms(
      ["exec.cap.query", "file.read"],
      ["custom.scope", "file.read", "zeta.scope"]
    )

    expect(merged).toEqual(["file.read", "exec.cap.query", "custom.scope", "zeta.scope"])
  })

  it("returns the documented preset for built-in roles", () => {
    expect(rolePresetPerms("superadmin")).toEqual(["*"])
    expect(rolePresetPerms("admin")).toContain("auth.register.approve")
    expect(rolePresetPerms("node")).toContain("exec.call")
    expect(rolePresetPerms("observer")).toBeNull()
  })

  it("exposes flattened options and permission metadata for dialog editors", () => {
    expect(accessPolicyPermissionOptions.find((item) => item.perm === "file.read")).toMatchObject({
      perm: "file.read",
      groupId: "file",
      groupLabel: "File"
    })

    expect(findPermissionCatalogItem("auth.register.approve")).toMatchObject({
      perm: "auth.register.approve",
      groupId: "auth",
      label: "auth.register.approve"
    })

    expect(findPermissionCatalogItem("custom.scope")).toBeNull()
  })
})
