// Context: covers the detached stream source window behavior in the Win frontend.

// @vitest-environment jsdom

import { defineComponent, nextTick } from "vue";
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { setLocale } from "@/i18n";

const hoisted = vi.hoisted(() => {
  const publishCaptureChunk = vi.fn(async (input: any) => ({
    sourceId: input.sourceId,
    sent: input.final ? 0 : input.deliveryIds.length,
    deliveryIds: input.deliveryIds,
  }));

  return {
    loadHomeState: vi.fn(async () => ({ nodeId: 7, hubId: 9 })),
    publishCaptureChunk,
    streamStore: {
      setIdentity: vi.fn(),
      loadPrefs: vi.fn(async () => undefined),
      loadDeliveries: vi.fn(async () => undefined),
      sourceById: vi.fn(() => ({
        sourceId: "source-video",
        producer: 7,
        name: "Desktop Capture",
        kind: "video",
        contentType: "video/webm",
        mode: "bounded",
        unitMode: "chunk",
        tags: [],
        metadataRaw: "",
        inputKind: "desktop",
        filePath: "",
      })),
      deliveriesForSource: vi.fn(() => [
        {
          deliveryId: "delivery-1",
          consumer: 9,
          consumerId: "consumer-1",
          state: "active",
        },
      ]),
      publishText: vi.fn(async () => ({
        sourceId: "source-video",
        sent: 1,
        deliveryIds: ["delivery-1"],
      })),
      pickMediaFile: vi.fn(async () => null),
      updateSourceInput: vi.fn(async () => undefined),
      publishCaptureChunk,
    },
    toastStore: {
      error: vi.fn(),
      errorOf: vi.fn(),
      success: vi.fn(),
      warn: vi.fn(),
    },
  };
});

const loadHomeState = hoisted.loadHomeState;
const publishCaptureChunk = hoisted.publishCaptureChunk;
const streamStore = hoisted.streamStore;
const toastStore = hoisted.toastStore;

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {
      sourceId: "source-video",
    },
  }),
}));

vi.mock("@/stores/session", () => ({
  useSessionStore: () => ({
    auth: {
      nodeId: 7,
      hubId: 9,
    },
  }),
}));

vi.mock("@/stores/stream", () => ({
  useStreamStore: () => hoisted.streamStore,
}));

vi.mock("@/stores/toast", () => ({
  useToastStore: () => hoisted.toastStore,
}));

vi.mock("../../wailsjs/go/main/App", () => ({
  HomeState: hoisted.loadHomeState,
}));

import StreamSourceWindow from "./StreamSourceWindow.vue";

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

const flushAsync = async () => {
  await Promise.resolve();
  await Promise.resolve();
  await nextTick();
};

class FakeMediaRecorder {
  static isTypeSupported = vi.fn(() => true);
  state = "inactive";
  ondataavailable: ((event: { data: Blob }) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onstop: ((event: Event) => void) | null = null;

  constructor(_stream: MediaStream, _options?: MediaRecorderOptions) {}

  start() {
    this.state = "recording";
  }

  requestData() {
    this.ondataavailable?.({ data: new Blob([new Uint8Array([1, 2, 3, 4])]) });
  }

  stop() {
    this.state = "inactive";
    this.onstop?.(new Event("stop"));
  }
}

describe("StreamSourceWindow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setLocale("en");
    loadHomeState.mockResolvedValue({ nodeId: 7, hubId: 9 });
    publishCaptureChunk.mockResolvedValue({
      sourceId: "source-video",
      sent: 1,
      deliveryIds: ["delivery-1"],
    });
    vi.stubGlobal("MediaRecorder", FakeMediaRecorder);
    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(
      undefined as never,
    );
    const track = { stop: vi.fn(), onended: null as null | (() => void) };
    Object.defineProperty(window.navigator, "mediaDevices", {
      configurable: true,
      value: {
        getDisplayMedia: vi.fn(async () => ({
          getTracks: () => [track],
          getVideoTracks: () => [track],
        })),
      },
    });
  });

  it("starts desktop capture and publishes a session-start chunk followed by a final chunk on stop", async () => {
    const wrapper = mount(StreamSourceWindow, {
      global: {
        stubs: {
          CardHeader: CardHeaderStub,
          Button: ButtonStub,
          Badge: BadgeStub,
        },
      },
    });

    await flushAsync();
    await flushAsync();

    expect(wrapper.text()).toContain(
      "Desktop capture sends live media chunks directly into the current active deliveries.",
    );

    await wrapper.get("[data-stream-source-start-capture]").trigger("click");
    await flushAsync();

    await wrapper.get("[data-stream-source-stop-capture]").trigger("click");
    await flushAsync();
    await flushAsync();

    expect(
      window.navigator.mediaDevices.getDisplayMedia as unknown as ReturnType<
        typeof vi.fn
      >,
    ).toHaveBeenCalledWith({
      video: true,
      audio: false,
    });
    expect(publishCaptureChunk).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        sourceId: "source-video",
        deliveryIds: ["delivery-1"],
        payload: [1, 2, 3, 4],
        sessionStart: true,
      }),
    );
    expect(publishCaptureChunk).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        sourceId: "source-video",
        deliveryIds: ["delivery-1"],
        final: true,
      }),
    );
  });
});
