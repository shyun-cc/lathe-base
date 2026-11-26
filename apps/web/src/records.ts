import { quantity } from './api'
import type { ID, Row } from './api'

export interface Column {
  key: string
  label: string
  reference?: string
}

export interface Resource {
  label: string
  path: string
  name: string
  columns: Column[]
  master?: boolean
  role?: 'is_customer' | 'is_supplier'
  printType?: string
}

export const column = (key: string, label: string, reference?: string): Column => ({ key, label, reference })

const masterColumns = [column('code', '编码'), column('name', '名称'), column('active', '状态')]
export const resources: Record<string, Resource> = {
  customers: { label: '客户', path: 'parties', name: 'name', columns: masterColumns, master: true, role: 'is_customer' },
  suppliers: { label: '供应商', path: 'parties', name: 'name', columns: masterColumns, master: true, role: 'is_supplier' },
  units: { label: '计量单位', path: 'units', name: 'name', columns: masterColumns, master: true },
  products: { label: '物料', path: 'products', name: 'name', columns: [...masterColumns.slice(0, 2), column('unit_id', '单位', 'units'), masterColumns[2]], master: true },
  operations: { label: '工序', path: 'operations', name: 'name', columns: masterColumns, master: true },
  employees: { label: '员工', path: 'employees', name: 'name', columns: masterColumns, master: true },
  sales: {
    label: '销售订单', path: 'sales-orders', name: 'order_no', printType: 'sales-order',
    columns: [column('order_no', '订单编号'), column('customer_id', '客户', 'parties'), column('ordered_on', '订单日期'), column('currency', '币种'), column('status', '状态')],
  },
  production: {
    label: '生产工单', path: 'work-orders', name: 'work_order_no', printType: 'work-order',
    columns: [column('work_order_no', '工单编号'), column('quantity_milli', '生产数量'), column('due_on', '交期'), column('status', '状态')],
  },
  outsource: {
    label: '委外加工', path: 'outsource-jobs', name: 'job_no',
    columns: [column('job_no', '委外单号'), column('supplier_id', '供应商', 'parties'), column('work_order_id', '工单', 'work-orders'), column('operation_name', '工序'), column('quantity_milli', '发出数量'), column('status', '状态')],
  },
  shipping: {
    label: '发货单', path: 'shipments', name: 'shipment_no', printType: 'shipment',
    columns: [column('shipment_no', '发货单号'), column('order_id', '销售订单', 'sales-orders'), column('customer_id', '客户', 'parties'), column('dispatched_on', '发货日期'), column('status', '状态')],
  },
  statements: {
    label: '往来对账', path: 'statements', name: 'statement_no',
    columns: [column('statement_no', '对账单号'), column('party_id', '往来方', 'parties'), column('direction', '方向'), column('currency', '币种'), column('amount_minor', '金额（最小单位）'), column('status', '状态')],
  },
  cash: {
    label: '收付款', path: 'cash-entries', name: 'entry_no',
    columns: [column('entry_no', '流水编号'), column('party_id', '往来方', 'parties'), column('direction', '方向'), column('booked_on', '入账日期'), column('amount_minor', '金额（最小单位）'), column('status', '状态')],
  },
  invoices: {
    label: '发票登记', path: 'invoices', name: 'invoice_no',
    columns: [column('invoice_no', '发票号码'), column('party_id', '往来方', 'parties'), column('direction', '方向'), column('invoiced_on', '开票日期'), column('gross_amount_minor', '含税金额（最小单位）'), column('status', '状态')],
  },
}

export const groups = [
  { label: '业务流转', items: ['sales', 'production', 'outsource', 'shipping'] },
  { label: '往来结算', items: ['statements', 'cash', 'invoices'] },
  { label: '基础档案', items: ['customers', 'suppliers', 'products', 'units', 'operations', 'employees'] },
  { label: '配置', items: ['templates'] },
]

export const statuses: Record<string, string> = {
  draft: '草稿', confirmed: '已确认', planned: '待下达', released: '已下达',
  in_progress: '生产中', completed: '已完工', sent: '已发出', closed: '已结清',
  dispatched: '已发货', delivered: '已送达', cancelled: '已取消', void: '已作废',
  posted: '已登记', registered: '已登记', receivable: '应收', payable: '应付',
  issued: '销项', received: '进项',
}

export function describe(row: Row, path: string) {
  const resource = Object.values(resources).find(item => item.path === path)
  return row.code ? `${row.code} · ${row.name}` : String(row[resource?.name ?? 'id'] ?? row.id)
}

export function display(row: Row, field: Column, references: Record<string, Row[]>) {
  const value = row[field.key]
  if (value === null || value === undefined || value === '') return '—'
  if (field.key === 'active') return value ? '启用' : '停用'
  if (field.key === 'status' || field.key === 'direction') return statuses[String(value)] ?? String(value)
  if (field.key.endsWith('_milli')) return quantity(value as ID)
  if (field.reference) {
    const path = field.reference
    const record = references[path]?.find(item => String(item.id) === String(value))
    if (record) return describe(record, path)
  }
  return String(value)
}
