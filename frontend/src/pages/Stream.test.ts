// @vitest-environment jsdom

import { defineComponent, nextTick, reactive } from "vue";
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { setLocale } from "@/i18n";

const streamState = reactive({
  activeTab: "source",
  targetId: "",
  selfNodeId: 0,
  defaultTargetId: 0,
  localSources: [] as Array<{
    sourceId: string;
    producer: number;
    name: string;
    kind: string;
    contentType: string;
    mode: string;
    unitMode: string;
    tags: string[];
    metadataRaw: string;
    inputKind: string;
    filePath: string;
  }>,
  localConsumers: [] as Array<{
    consumerId: string;
    consumer: number;
    name: string;
    kind: string;
    contentType: string;
    tags: string[];
    metadataRaw: string;
  }>,
  sources: [] as Array<{
    sourceId: string;
    producer: number;
    name: string;
    kind: string;
    contentType: string;
    mode: string;
    unitMode: string;
    tags: string[];
    metadataRaw: string;
    inputKind: string;
    filePath: string;
  }>,
  consumers: [] as Array<{
    consumerId: string;
    consumer: number;
    name: string;
    kind: string;
    contentType: string;
    tags: string[];
    metadataRaw: string;
  }>,
  deliveries: [] as Array<{
    deliveryId: string;
    producer: number;
    consumer: number;
    consumerId: string;
    sourceId: string;
    kind: string;
    state: string;
    bytesIn: number;
    framesIn: number;
    updatedAt: string;
  }>,
  selectedSourceId: "",
  selectedConsumerId: "",
  selectedDeliveryId: "",
  lastSyncAt: "",
  lastEventAt: "",
  textFramesByDelivery: {} as Record<
    string,
    Array<{
      text: string;
      position: number;
      updatedAt: string;
      deliveryId: string;
    }>
  >,
  statsByDelivery: {} as Record<string, unknown>,
  mediaByDelivery: {} as Record<string, unknown>,
});

const resetStreamState = () => {
  streamState.activeTab = "source";
  streamState.targetId = "";
  streamState.selfNodeId = 0;
  streamState.defaultTargetId = 0;
  streamState.localSources = [];
  streamState.localConsumers = [];
  streamState.sources = [];
  streamState.consumers = [];
  streamState.deliveries = [];
  streamState.selectedSourceId = "";
  streamState.selectedConsumerId = "";
  streamState.selectedDeliveryId = "";
  streamState.lastSyncAt = "";
  streamState.lastEventAt = "";
  streamState.textFramesByDelivery = {};
  streamState.statsByDelivery = {};
  streamState.mediaByDelivery = {};
};

const streamStore = {
  state: streamState,
  setIdentity: vi.fn((nodeId: number, hubId: number) => {
    streamState.selfNodeId = nodeId;
    streamState.defaultTargetId = hubId;
  }),
  setTargetId: vi.fn((value: string) => {
    streamState.targetId = String(value ?? "").trim();
  }),
  setActiveTab: vi.fn((value: "source" | "consumer" | "control") => {
    streamState.activeTab = value;
  }),
  loadPrefs: vi.fn(async () => undefined),
  loadDeliveries: vi.fn(async () => streamState.deliveries),
  loadMedia: vi.fn(async () => streamState.mediaByDelivery),
  listSources: vi.fn(
    async (producer: string, _kind = "", _tag = "", scope = "catalog") => {
      if (scope === "catalog") {
        streamState.sources = [
          {
            sourceId: "remote-source-1",
            producer: Number(producer || 12) || 12,
            name: "Remote Source",
            kind: "text",
            contentType: "text/plain",
            mode: "live",
            unitMode: "frame",
            tags: ["alpha"],
            metadataRaw: "",
            inputKind: "",
            filePath: "",
          },
        ];
        return streamState.sources;
      }
      return streamState.localSources;
    },
  ),
  listConsumers: vi.fn(
    async (_consumer: string, _kind = "", _tag = "", scope = "catalog") => {
      if (scope === "catalog") {
        streamState.consumers = [
          {
            consumerId: "remote-consumer-1",
            consumer: 21,
            name: "Remote Consumer",
            kind: "text",
            contentType: "text/plain",
            tags: [],
            metadataRaw: "",
          },
        ];
        return streamState.consumers;
      }
      return streamState.localConsumers;
    },
  ),
  announceSource: vi.fn(
    async (draft: {
      name: string;
      kind: string;
      contentType: string;
      mode: string;
      unitMode: string;
      tagsText: string;
      metadataText: string;
      inputKind?: string;
      filePath?: string;
    }) => {
      const source = {
        sourceId: `source-${streamState.localSources.length + 1}`,
        producer: streamState.selfNodeId || 7,
        name: draft.name,
        kind: draft.kind,
        contentType: draft.contentType,
        mode: draft.mode,
        unitMode: draft.unitMode,
        tags: [],
        metadataRaw: draft.metadataText,
        inputKind: draft.inputKind ?? "",
        filePath: draft.filePath ?? "",
      };
      streamState.localSources = [source, ...streamState.localSources];
      return source;
    },
  ),
  announceConsumer: vi.fn(
    async (draft: {
      name: string;
      kind: string;
      contentType: string;
      metadataText: string;
    }) => {
      const consumer = {
        consumerId: `consumer-${streamState.localConsumers.length + 1}`,
        consumer: streamState.selfNodeId || 7,
        name: draft.name,
        kind: draft.kind,
        contentType: draft.contentType,
        tags: [],
        metadataRaw: draft.metadataText,
      };
      streamState.localConsumers = [consumer, ...streamState.localConsumers];
      return consumer;
    },
  ),
  publishText: vi.fn(async () => ({
    sourceId: "source-1",
    sent: 1,
    deliveryIds: ["delivery-1"],
  })),
  pickMediaFile: vi.fn(async () => null),
  subscribe: vi.fn(async () => undefined),
  connect: vi.fn(async () => undefined),
  disconnect: vi.fn(async () => undefined),
  unsubscribe: vi.fn(async () => undefined),
  signal: vi.fn(async () => undefined),
  updateSourceInput: vi.fn(async () => undefined),
  withdrawSource: vi.fn(async () => undefined),
  withdrawConsumer: vi.fn(async () => undefined),
  sourceById: vi.fn((sourceId: string, scope = "any") => {
    const normalized = String(sourceId ?? "").trim();
    if (scope === "local")
      return (
        streamState.localSources.find((item) => item.sourceId === normalized) ??
        null
      );
    if (scope === "catalog")
      return (
        streamState.sources.find((item) => item.sourceId === normalized) ?? null
      );
    return (
      streamState.localSources.find((item) => item.sourceId === normalized) ??
      streamState.sources.find((item) => item.sourceId === normalized) ??
      null
    );
  }),
  consumerById: vi.fn((consumerId: string, scope = "any") => {
    const normalized = String(consumerId ?? "").trim();
    if (scope === "local")
      return (
        streamState.localConsumers.find(
          (item) => item.consumerId === normalized,
        ) ?? null
      );
    if (scope === "catalog")
      return (
        streamState.consumers.find((item) => item.consumerId === normalized) ??
        null
      );
    return (
      streamState.localConsumers.find(
        (item) => item.consumerId === normalized,
      ) ??
      streamState.consumers.find((item) => item.consumerId === normalized) ??
      null
    );
  }),
  deliveriesForSource: vi.fn((sourceId: string) =>
    streamState.deliveries.filter((item) => item.sourceId === sourceId),
  ),
  deliveriesForConsumer: vi.fn((consumerId: string) =>
    streamState.deliveries.filter((item) => item.consumerId === consumerId),
  ),
  selectSource: vi.fn((sourceId: string) => {
    streamState.selectedSourceId = sourceId;
  }),
  selectConsumer: vi.fn((consumerId: string) => {
    streamState.selectedConsumerId = consumerId;
  }),
  selectDelivery: vi.fn((deliveryId: string) => {
    streamState.selectedDeliveryId = deliveryId;
  }),
  textFramesFor: vi.fn(
    (deliveryId: string) => streamState.textFramesByDelivery[deliveryId] ?? [],
  ),
  statsFor: vi.fn(() => null),
  mediaForDelivery: vi.fn(
    (deliveryId: string) => streamState.mediaByDelivery[deliveryId] ?? null,
  ),
};

const sessionStore = reactive({
  auth: {
    nodeId: 7,
    hubId: 9,
  },
});

const toastStore = {
  success: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
  errorOf: vi.fn(),
  info: vi.fn(),
};

const windowOpenSpy = vi.spyOn(window, "open").mockImplementation(
  () =>
    ({
      focus: vi.fn(),
    }) as unknown as Window,
);

vi.mock("@/stores/stream", () => ({
  streamKinds: ["music", "video", "text", "custom"],
  useStreamStore: () => streamStore,
}));

vi.mock("@/stores/session", () => ({
  useSessionStore: () => sessionStore,
}));

vi.mock("@/stores/toast", () => ({
  useToastStore: () => toastStore,
}));

import Stream from "./Stream.vue";

const PageHeroStub = defineComponent({
  props: {
    description: { type: String, default: "" },
  },
  template: `<section><p v-if="description">{{ description }}</p><slot name="actions" /></section>`,
});

const CardHeaderStub = defineComponent({
  props: {
    title: { type: String, default: "" },
    description: { type: String, default: "" },
  },
  template: `<section><h2>{{ title }}</h2><p v-if="description">{{ description }}</p><slot name="actions" /></section>`,
});

const ButtonStub = defineComponent({
  inheritAttrs: false,
  emits: ["click"],
  template: `<button type="button" v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`,
});

const BadgeStub = defineComponent({
  inheritAttrs: false,
  template: `<span v-bind="$attrs"><slot /></span>`,
});

const OverlayStub = defineComponent({
  props: {
    open: { type: Boolean, default: false },
  },
  template: `<div v-if="open"><slot /></div>`,
});

const mountPage = () =>
  mount(Stream, {
    global: {
      stubs: {
        PageHero: PageHeroStub,
        CardHeader: CardHeaderStub,
        Button: ButtonStub,
        Badge: BadgeStub,
        Overlay: OverlayStub,
      },
    },
  });

describe("Stream page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setLocale("zh-CN");
    resetStreamState();
    sessionStore.auth.nodeId = 7;
    sessionStore.auth.hubId = 9;
  });

  it("renders the compact tabbed stream page and removes the old summary cards", async () => {
    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    expect(streamStore.setIdentity).toHaveBeenCalledWith(7, 9);
    expect(wrapper.text()).toContain("源");
    expect(wrapper.text()).toContain("消费者");
    expect(wrapper.text()).toContain("控制");
    expect(wrapper.text()).not.toContain("已保存 Source");
    expect(wrapper.text()).not.toContain("已保存 Consumer");
    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(false);

    await wrapper.get("[data-stream-open-source]").trigger("click");
    await nextTick();

    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(true);
  });

  it("creates a local source from the dialog and opens the dedicated source input window", async () => {
    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    await wrapper.get("[data-stream-open-source]").trigger("click");
    await nextTick();
    await wrapper.get("#stream-source-name").setValue("Local Text Source");
    await wrapper.get("[data-stream-submit-source]").trigger("click");
    await Promise.resolve();
    await nextTick();

    expect(streamStore.announceSource).toHaveBeenCalledTimes(1);
    expect(wrapper.find("[data-stream-source-dialog]").exists()).toBe(false);
    expect(streamState.localSources).toHaveLength(1);

    await wrapper.get("[data-stream-open-source-window]").trigger("click");

    expect(windowOpenSpy).toHaveBeenCalledWith(
      expect.stringContaining("#/stream-source-window?sourceId=source-1"),
      expect.stringContaining("stream_source_source-1_"),
      "width=1100,height=780",
    );
  });

  it("shows desktop capture configuration for video sources in the source dialog", async () => {
    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    await wrapper.get("[data-stream-open-source]").trigger("click");
    await nextTick();
    await wrapper.get("#stream-source-name").setValue("Desktop Capture");
    await wrapper.get("[data-stream-source-kind]").setValue("video");
    await nextTick();
    await wrapper.get("[data-stream-source-input-mode]").setValue("desktop");
    await nextTick();

    expect(wrapper.find("[data-stream-source-desktop-config]").exists()).toBe(
      true,
    );
    expect(wrapper.find("[data-stream-source-file-config]").exists()).toBe(
      false,
    );
  });

  it("opens the subscribe dialog from a local consumer and subscribes the selected source", async () => {
    streamState.localConsumers = [
      {
        consumerId: "consumer-1",
        consumer: 7,
        name: "Local Consumer",
        kind: "text",
        contentType: "text/plain",
        tags: [],
        metadataRaw: "",
      },
    ];

    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    await wrapper.get("[data-stream-tab='consumer']").trigger("click");
    await nextTick();
    await wrapper.get("[data-stream-open-subscribe]").trigger("click");
    await Promise.resolve();
    await nextTick();

    expect(wrapper.find("[data-stream-subscribe-dialog]").exists()).toBe(true);
    expect(streamStore.listSources).toHaveBeenCalled();

    await wrapper.get("[data-stream-subscribe-source-row]").trigger("click");
    await wrapper.get("[data-stream-submit-subscribe]").trigger("click");
    await Promise.resolve();
    await nextTick();

    expect(streamStore.subscribe).toHaveBeenCalledWith({
      producer: streamState.sources[0].producer,
      sourceId: streamState.sources[0].sourceId,
      consumerId: "consumer-1",
    });
  });

  it("opens the control pair picker and connects the selected source and consumer", async () => {
    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    await wrapper.get("[data-stream-tab='control']").trigger("click");
    await nextTick();

    expect(wrapper.text()).not.toContain("Source Catalog");
    expect(wrapper.text()).not.toContain("Consumer Catalog");

    await wrapper.get("[data-stream-open-control-picker]").trigger("click");
    await Promise.resolve();
    await nextTick();

    expect(wrapper.find("[data-stream-control-dialog]").exists()).toBe(true);

    await wrapper.get("[data-stream-control-source-row]").trigger("click");
    await wrapper.get("[data-stream-control-consumer-row]").trigger("click");
    await wrapper.get("[data-stream-submit-control-connect]").trigger("click");
    await Promise.resolve();
    await nextTick();

    expect(streamStore.connect).toHaveBeenCalledWith({
      producer: streamState.sources[0].producer,
      sourceId: streamState.sources[0].sourceId,
      consumer: streamState.consumers[0].consumer,
      consumerId: streamState.consumers[0].consumerId,
    });
    expect(wrapper.find("[data-stream-control-dialog]").exists()).toBe(false);
  });

  it("opens a dedicated delivery output window from the control tab", async () => {
    streamState.deliveries = [
      {
        deliveryId: "delivery-1",
        producer: 7,
        consumer: 9,
        consumerId: "consumer-1",
        sourceId: "source-1",
        kind: "text",
        state: "active",
        bytesIn: 12,
        framesIn: 3,
        updatedAt: "2026-03-31T12:00:00Z",
      },
    ];

    const wrapper = mountPage();

    await Promise.resolve();
    await nextTick();

    await wrapper.get("[data-stream-tab='control']").trigger("click");
    await nextTick();
    await wrapper.get("[data-stream-open-delivery-window]").trigger("click");

    expect(streamStore.selectDelivery).toHaveBeenCalledWith("delivery-1");
    expect(windowOpenSpy).toHaveBeenCalledWith(
      expect.stringContaining("#/stream-delivery-window?deliveryId=delivery-1"),
      expect.stringContaining("stream_delivery_delivery-1_"),
      "width=1180,height=820",
    );
  });
});
