// Context: assembles the i18n catalog and exports the translator used across the Win frontend.

import { computed, reactive } from "vue"
import { messages } from "./messages"
import type { AppLocale, TranslationParams } from "./types"

export type { AppLocale } from "./types"

export const defaultLocale: AppLocale = "en"

const state = reactive<{ locale: AppLocale }>({
  locale: defaultLocale
})

export const normalizeLocale = (value: unknown): AppLocale => {
  return String(value ?? "").trim() === "zh-CN" ? "zh-CN" : "en"
}

const interpolate = (message: string, params?: TranslationParams) => {
  if (!params) return message
  return message.replace(/\{(\w+)\}/g, (_, key: string) => {
    const value = params[key]
    return value == null ? "" : String(value)
  })
}

const resolveMessage = (locale: AppLocale, key: string) => {
  return messages[locale]?.[key] ?? messages.en[key] ?? key
}

export const setLocale = (value: unknown) => {
  const next = normalizeLocale(value)
  state.locale = next
  if (typeof document !== "undefined") {
    document.documentElement.lang = next
  }
  return next
}

export const t = (key: string, params?: TranslationParams) => {
  return interpolate(resolveMessage(state.locale, key), params)
}

export const useI18n = () => {
  return {
    locale: computed(() => state.locale),
    setLocale,
    t
  }
}

export const currentLocale = () => state.locale
