// Context: covers the flow field binding dialog panel behavior used by the Flow editor.

// @vitest-environment jsdom

import { defineComponent, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it } from "vitest"
import { setLocale } from "@/i18n"
import FlowFieldBindingDialog from "./FlowFieldBindingDialog.vue"
import type { VisualFieldModel } from "@/stores/flow"

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" },
    titleId: { type: String, default: "" },
    descriptionId: { type: String, default: "" }
  },
  template: `
    <div>
      <h2 :id="titleId">{{ title }}</h2>
      <p v-if="description" :id="descriptionId">{{ description }}</p>
    </div>
  `
})

const ButtonStub = defineComponent({
  props: {
    disabled: { type: Boolean, default: false }
  },
  emits: ["click"],
  template: `<button type="button" :disabled="disabled" @click="$emit('click', $event)"><slot /></button>`
})

const OverlayStub = defineComponent({
  props: {
    open: { type: Boolean, default: false }
  },
  emits: ["close"],
  template: `<div v-if="open"><slot /></div>`
})

const activeBindingField: VisualFieldModel = {
  schema: {
    key: "name",
    label: "Name",
    pointer: "/name",
    control: "text",
    bindable: true
  },
  state: {
    mode: "literal",
    literalValue: "",
    binding: null
  },
  bindingSummary: ""
}

describe("FlowFieldBindingDialog", () => {
  beforeEach(() => {
    setLocale("en")
  })

  it("renders flow local var binding controls and requires a variable name", async () => {
    const fieldBindingDraft = reactive({
      sourceKind: "flow_var" as const,
      nodeId: "",
      path: "",
      field: "",
      name: "",
      required: false
    })

    const wrapper = mount(FlowFieldBindingDialog, {
      props: {
        open: true,
        activeBindingField,
        bindableAncestorNodeOptions: ["node-1"],
        fieldBindingDraft
      },
      global: {
        stubs: {
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Overlay: OverlayStub
        }
      }
    })

    expect(wrapper.text()).toContain("Flow Local Var")
    expect(wrapper.text()).toContain("This is not varstore.")
    expect(wrapper.findAll("button").at(-1)?.text()).toContain("Apply Binding")
    expect(wrapper.findAll("button").at(-1)?.attributes("disabled")).toBeDefined()

    await wrapper.get("#flow-field-binding-flow-var-name").setValue("session_token")

    const buttons = wrapper.findAll("button")
    expect(buttons.at(-1)?.attributes("disabled")).toBeUndefined()
  })
})
