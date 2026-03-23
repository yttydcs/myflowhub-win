export type FormInputValue = string | number | boolean | null | undefined

type NumericInputOptions<TBlank = never> = {
  allowBlank?: boolean
  blankValue?: TBlank
  requiredMessage?: string
  invalidMessage: string
  min?: number
  max?: number
}

export const normalizeFormInputText = (value: FormInputValue) =>
  value === undefined || value === null ? "" : String(value)

export const isBlankFormInput = (value: FormInputValue) => !normalizeFormInputText(value).trim()

const ensureBlankValue = <TBlank>(options: NumericInputOptions<TBlank>) => {
  if (options.allowBlank) {
    return options.blankValue as TBlank
  }
  throw new Error(options.requiredMessage ?? options.invalidMessage)
}

const ensureInRange = (value: number, options: NumericInputOptions<unknown>) => {
  if (options.min !== undefined && value < options.min) {
    throw new Error(options.invalidMessage)
  }
  if (options.max !== undefined && value > options.max) {
    throw new Error(options.invalidMessage)
  }
  return value
}

export const parseIntegerInput = <TBlank = never>(value: FormInputValue, options: NumericInputOptions<TBlank>) => {
  const trimmed = normalizeFormInputText(value).trim()
  if (!trimmed) {
    return ensureBlankValue(options)
  }
  const parsed = Number.parseInt(trimmed, 10)
  if (Number.isNaN(parsed)) {
    throw new Error(options.invalidMessage)
  }
  return ensureInRange(parsed, options)
}

export const parseFloatInput = <TBlank = never>(value: FormInputValue, options: NumericInputOptions<TBlank>) => {
  const trimmed = normalizeFormInputText(value).trim()
  if (!trimmed) {
    return ensureBlankValue(options)
  }
  const parsed = Number.parseFloat(trimmed)
  if (!Number.isFinite(parsed)) {
    throw new Error(options.invalidMessage)
  }
  return ensureInRange(parsed, options)
}

export const parseNumberInput = <TBlank = never>(value: FormInputValue, options: NumericInputOptions<TBlank>) => {
  const trimmed = normalizeFormInputText(value).trim()
  if (!trimmed) {
    return ensureBlankValue(options)
  }
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed)) {
    throw new Error(options.invalidMessage)
  }
  return ensureInRange(parsed, options)
}
