// Context: contains shared showcase layout helpers used by the Win frontend.

export type ColumnsLayoutInput = {
  maxColumns: number
  minColumnWidth: number
  gap: number
}

export const computeColumnsCount = (containerWidth: number, layout: ColumnsLayoutInput): number => {
  const width = Number.isFinite(containerWidth) ? containerWidth : 0
  const gap = Number.isFinite(layout.gap) ? Math.max(0, Math.floor(layout.gap)) : 0
  const minWidth = Number.isFinite(layout.minColumnWidth)
    ? Math.max(1, Math.floor(layout.minColumnWidth))
    : 1
  const maxColumns = Number.isFinite(layout.maxColumns)
    ? Math.max(1, Math.floor(layout.maxColumns))
    : 1

  if (width <= 0) return 1
  const raw = Math.floor((width + gap) / (minWidth + gap))
  const resolved = raw > 0 ? raw : 1
  return Math.min(maxColumns, Math.max(1, resolved))
}

export const clampColSpan = (colSpan: number, columns: number): number => {
  const cols = Number.isFinite(columns) ? Math.max(1, Math.floor(columns)) : 1
  const spanRaw = Number.isFinite(colSpan) ? Math.floor(colSpan) : 1
  const span = spanRaw > 0 ? spanRaw : 1
  return Math.min(cols, Math.max(1, span))
}

