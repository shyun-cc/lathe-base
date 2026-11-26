import type { ID } from './api'

export interface PrintField { key: string; label: string }
export interface PrintCell { label: string; field: string; text: string; span: number }
export interface PrintColumn { field: string; label: string; width: number; align: 'left' | 'center' | 'right' }
export interface Configuration {
  title: string
  paper: 'A4' | 'A5'
  orientation: 'portrait' | 'landscape'
  columnsPerRow: number
  header: PrintCell[]
  columns: PrintColumn[]
  footer: PrintCell[]
}
export interface Definition { label: string; fields: PrintField[]; detail: PrintField[]; default: Configuration }
export interface Template {
  id: ID
  code: string
  name: string
  document_type: string
  version: number
  configuration: Configuration
  active: boolean
}
type PrintRow = Record<string, string | null>
export interface PrintDocument { fields: PrintRow; rows: PrintRow[] }

function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('配置必须是 JSON 对象。')
  return value as Record<string, unknown>
}

function integer(value: unknown, min: number, max: number): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= min && value <= max
}

function bounded(value: unknown, max: number): value is string {
  return typeof value === 'string' && [...value].length <= max
}

export function parseConfiguration(value: unknown, definition: Definition): Configuration {
  const { title, paper, orientation, columnsPerRow, header, columns, footer } = object(value)
  if (!bounded(title, 80) || !title.trim() || (paper !== 'A4' && paper !== 'A5') ||
      (orientation !== 'portrait' && orientation !== 'landscape') || !integer(columnsPerRow, 1, 4)) {
    throw new Error('请输入标题，并选择纸张、方向和每行栏数（1–4）。')
  }
  function cells(input: unknown) {
    if (!Array.isArray(input) || input.length > 24) throw new Error('表头和表底栏各可配置 0–24 项。')
    return input.map(item => {
      const { label, field, text, span } = object(item)
      if (!bounded(label, 80) || !bounded(text, 500) || typeof field !== 'string' ||
          !integer(span, 1, Number(columnsPerRow)) ||
          (field ? Boolean(text) || !definition.fields.some(option => option.key === field) : !label.trim() && !text.trim())) {
        throw new Error('请检查表头或表底栏的字段、文字和占栏数。')
      }
      return { label, field, text, span }
    })
  }
  if (!Array.isArray(columns) || columns.length < 1 || columns.length > 12) throw new Error('请配置 1–12 个明细列。')
  return {
    title, paper, orientation, columnsPerRow,
    header: cells(header), footer: cells(footer),
    columns: columns.map(item => {
      const { label, field, width, align } = object(item)
      if (!bounded(label, 80) || !label.trim() || typeof field !== 'string' ||
          !definition.detail.some(option => option.key === field) || !integer(width, 1, 100) ||
          (align !== 'left' && align !== 'center' && align !== 'right')) {
        throw new Error('请检查明细列的字段、列名和相对宽度（1–100）。')
      }
      return { label, field, width, align }
    }),
  }
}

export function sample(definition: Definition): PrintDocument {
  const fields: PrintRow = {
    number: 'DEMO-001', party: '示例客户', date: '2026-01-15', currency: 'CNY', total: '12000', order: 'SO-DEMO-001',
  }
  const detail: PrintRow = { product: '示例物料', unit: '件', quantity: '12', amount: '12000', due: '2026-01-20' }
  return {
    fields: Object.fromEntries(definition.fields.map(field => [field.key, fields[field.key] ?? field.label])),
    rows: [Object.fromEntries(definition.detail.map(field => [field.key, detail[field.key] ?? field.label]))],
  }
}
