import { describe, expect, it } from "vitest"

import { shellZhCN } from "./shell"

describe("shellZhCN", () => {
  it("renders the Authority group as access control in Chinese", () => {
    expect(shellZhCN["Authority"]).toBe("访问控制")
  })
})
