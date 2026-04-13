// Context: provides the shell test support code used by the Win desktop host.

import { describe, expect, it } from "vitest"

import { shellZhCN } from "./shell"

describe("shellZhCN", () => {
  it("renders the Authority group as access control in Chinese", () => {
    expect(shellZhCN["Authority"]).toBe("访问控制")
  })

  it("translates the Stream navigation and route copy", () => {
    expect(shellZhCN["Stream"]).toBe("流")
    expect(shellZhCN["Sources and deliveries"]).toBe("源端与传输")
    expect(shellZhCN["Query typed sources and consumers, connect deliveries, and inspect runtime traffic."]).toBe(
      "查询带类型的源端与消费端、连接传输，并查看运行时流量。"
    )
  })
})
