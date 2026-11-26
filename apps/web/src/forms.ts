import { dateToday } from './api'
import type { Row } from './api'
import type { Resource } from './records'

type InputValue = string | boolean
export type Values = Record<string, InputValue>
export interface Option { value: string; label: string }
export interface Field {
  key: string
  label: string
  type?: 'text' | 'date' | 'money' | 'quantity' | 'select' | 'boolean' | 'multiple'
  initial?: InputValue
  optional?: boolean
  allowZero?: boolean
  options?: Option[]
  source?: string | ((values: Values) => string)
  filter?: (row: Row, values: Values) => boolean
}
export interface FormSpec {
  title: string
  path: string
  method?: string
  fields: Field[]
  lines?: { key: string; label: string; fields: Field[] }
  fixed?: Record<string, unknown>
  result?: string
}

const text = (key: string, label: string, optional = false): Field => ({ key, label, optional })
const day = (key: string, label: string, optional = false): Field => ({
  key, label, type: 'date', initial: optional ? '' : dateToday(), optional,
})
const money = (key = 'amount_minor', label = '金额（最小单位）', allowZero = true): Field => ({
  key, label, type: 'money', allowZero,
})
const qty = (key = 'quantity_milli', label = '数量', allowZero = false): Field => ({ key, label, type: 'quantity', allowZero })
const ref = (key: string, label: string, source: Field['source'], filter?: Field['filter']): Field => ({
  key, label, type: 'select', source, filter,
})
const option = (value: string, label: string) => ({ value, label })
const currency = (): Field => ({ ...text('currency', '币种（三位代码）'), initial: 'CNY' })
const enabled = (row: Row) => row.active === true
const customer = (row: Row) => enabled(row) && row.is_customer === true
const supplier = (row: Row) => enabled(row) && row.is_supplier === true
const openWork = (row: Row) => row.status === 'released' || row.status === 'in_progress'
const direction = (): Field => ({
  key: 'direction', label: '方向', type: 'select', initial: 'receivable',
  options: [option('receivable', '应收'), option('payable', '应付')],
})
export function masterForm(resource: Resource, row?: Row): FormSpec {
  const fields = [text('code', '编码'), text('name', '名称')]
  if (resource.role) {
    fields.push(
      { key: 'is_customer', label: '客户', type: 'boolean', initial: resource.role === 'is_customer' },
      { key: 'is_supplier', label: '供应商', type: 'boolean', initial: resource.role === 'is_supplier' },
    )
  }
  if (resource.path === 'products') {
    fields.push(ref('unit_id', '计量单位', 'units', unit => enabled(unit) || String(unit.id) === String(row?.unit_id)))
  }
  fields.push({ key: 'active', label: '启用', type: 'boolean', initial: true })
  if (row) {
    for (const field of fields) {
      const value = row[field.key]
      if (value !== undefined && value !== null) field.initial = field.type === 'boolean' ? Boolean(value) : String(value)
    }
  }
  return { title: (row ? '编辑' : '新增') + resource.label, path: row ? `${resource.path}/${row.id}` : resource.path,
    method: row ? 'PUT' : 'POST', fields }
}

export function createForm(key: string): FormSpec | undefined {
  switch (key) {
    case 'sales': return {
      title: '新建销售订单', path: 'sales-orders', result: 'sales',
      fields: [text('order_no', '订单编号'), ref('customer_id', '客户', 'parties', customer), currency(), day('ordered_on', '订单日期')],
      lines: { key: 'lines', label: '订单明细', fields: [
        ref('product_id', '物料', 'products', enabled), qty(), money(), day('due_on', '交期', true),
      ] },
    }
    case 'outsource': return outsourceForm()
    case 'statements': return {
      title: '新建对账单', path: 'statements', result: 'statements',
      fields: [
        text('statement_no', '对账单号'), direction(), ref('party_id', '往来方', 'parties'),
        currency(), day('period_start', '账期起始'), day('period_end', '账期截止'),
        {
          key: 'source_ids', label: '对账来源', type: 'multiple',
          source: values => values.direction === 'payable' ? 'outsource-jobs' : 'shipments',
          filter: (row, values) => {
            const payable = values.direction === 'payable'
            const date = String(payable ? row.last_received_on : row.dispatched_on)
            return (payable ? row.status === 'closed' : ['dispatched', 'delivered'].includes(String(row.status))) &&
              String(payable ? row.supplier_id : row.customer_id) === values.party_id &&
              row.currency === String(values.currency).toUpperCase() &&
              (date >= String(values.period_start) && date <= String(values.period_end))
          },
        },
      ],
    }
    case 'cash': return {
      title: '登记收付款', path: 'cash-entries', result: 'cash',
      fields: [text('entry_no', '流水编号'), direction(), ref('party_id', '往来方', 'parties'),
        currency(), day('booked_on', '入账日期'), money('amount_minor', '金额（最小单位）', false),
        text('method', '收付方式'), text('reference', '参考号', true)],
    }
    case 'invoices': return {
      title: '登记发票', path: 'invoices', result: 'invoices',
      fields: [
        text('invoice_no', '发票号码'),
        { key: 'direction', label: '方向', type: 'select', initial: 'issued', options: [option('issued', '销项'), option('received', '进项')] },
        ref('party_id', '往来方', 'parties'), currency(), day('invoiced_on', '开票日期'),
        money('gross_amount_minor', '含税金额（最小单位）', false), money('tax_amount_minor', '税额（最小单位）'),
        { ...ref('statement_id', '关联对账单', 'statements', (row, values) =>
          row.status === 'confirmed' && String(row.party_id) === values.party_id &&
          row.direction === (values.direction === 'issued' ? 'receivable' : 'payable') &&
          row.currency === String(values.currency).toUpperCase()), optional: true },
        text('notes', '备注', true),
      ],
    }
  }
}

export function splitForm(line: Row): FormSpec {
  return {
    title: '拆分工单', path: `sales-order-lines/${line.id}/work-orders`, result: 'production', fields: [],
    lines: { key: 'batches', label: '工单批次', fields: [text('work_order_no', '工单编号'), qty(), day('due_on', '交期', true)] },
  }
}

export function shipmentForm(order: Row): FormSpec {
  const lines = order.lines as Row[]
  return {
    title: '新建发货单', path: 'shipments', result: 'shipping', fixed: { order_id: order.id },
    fields: [text('shipment_no', '发货单号')],
    lines: { key: 'lines', label: '发货明细', fields: [
      { key: 'sales_order_line_id', label: '订单明细', type: 'select',
        options: lines.map(line => option(String(line.id), `${line.line_no} · ${line.product_name}`)) },
      qty('quantity_milli', '本次发货数量'),
    ] },
  }
}

export function outsourceForm(work?: Row): FormSpec {
  return {
    title: '委外发出', path: 'outsource-jobs', result: 'outsource',
    fixed: work ? { work_order_id: work.id } : undefined,
    fields: [
      text('job_no', '委外单号'), ...(work ? [] : [ref('work_order_id', '生产工单', 'work-orders', openWork)]),
      ref('supplier_id', '供应商', 'parties', supplier),
      ref('operation_id', '工序', 'operations', enabled), qty(), currency(), money(),
      day('sent_on', '发出日期'), day('due_on', '预计回收日期', true),
    ],
  }
}

export function actionForms(key: string, row: Row): FormSpec[] {
  const id = row.id
  if (key === 'production' && openWork(row)) {
    return [
      { title: '登记报工', path: `work-orders/${id}/reports`, fields: [
        text('report_no', '报工编号'), ref('operation_id', '工序', 'operations', enabled),
        ref('employee_id', '员工', 'employees', enabled), day('reported_on', '报工日期'), qty(),
      ] },
      { title: '确认完工', path: `work-orders/${id}/complete`, fields: [ref('operation_id', '终道工序', 'operations')] },
      outsourceForm(row),
    ]
  }
  if (key === 'outsource' && row.status === 'sent') {
    return [{ title: '登记回收', path: `outsource-jobs/${id}/returns`, fields: [
      text('return_no', '回收单号'), day('received_on', '回收日期'),
      { ...qty('accepted_milli', '合格数量', true), initial: '0' },
      { ...qty('rejected_milli', '不合格数量', true), initial: '0' },
    ] }]
  }
  if (key === 'shipping' && row.status === 'draft') {
    return [{ title: '确认发货', path: `shipments/${id}/confirm`, fields: [day('dispatched_on', '发货日期')] }]
  }
  if (key === 'statements' && row.status === 'confirmed' && row.remaining_minor && Number(row.remaining_minor) > 0) {
    return [{ title: '收付款核销', path: 'allocations',
      fixed: { statement_id: id },
      fields: [
        ref('cash_entry_id', '收付款流水', 'cash-entries', cash =>
          cash.status === 'posted' && Number(cash.remaining_minor) > 0 && cash.party_id === row.party_id && cash.currency === row.currency && cash.direction === row.direction),
        money('amount_minor', '本次核销金额（最小单位）', false),
      ],
    }]
  }
  return []
}

export function sourceOf(field: Field, values: Values) {
  return typeof field.source === 'function' ? field.source(values) : field.source
}

export function defaults(fields: Field[]): Values {
  return Object.fromEntries(fields.map(field => [field.key, field.initial ?? (field.type === 'boolean' ? false : '')]))
}

export function encode(field: Field, value: InputValue): unknown {
  if (field.type === 'boolean') return Boolean(value)
  const input = String(value).trim()
  if (input === '' && field.optional) return undefined
  if (input === '') throw new Error('请填写' + field.label)
  if (field.type === 'multiple') {
    const ids = input.split(',')
    if (ids.length > 200) throw new Error('最多选择 200 张来源单据')
    return ids.map(id => BigInt(id))
  }
  if ((field.type === 'select' && field.source) || field.key === 'sales_order_line_id') return BigInt(input)
  if (field.type === 'quantity' || field.type === 'money') {
    const pattern = field.type === 'quantity' ? /^\d+(\.\d{1,3})?$/ : /^\d+$/
    if (!pattern.test(input)) throw new Error(field.type === 'quantity' ? '数量最多可填写三位小数' : '金额须为整数')
    const [whole, fraction = ''] = input.split('.')
    const integer = field.type === 'quantity' ? BigInt(whole) * 1000n + BigInt(fraction.padEnd(3, '0')) : BigInt(whole)
    if (integer > 9223372036854775807n || (!field.allowZero && integer === 0n)) throw new Error('数值超出范围')
    return integer
  }
  if (field.key === 'currency') {
    if (!/^[a-z]{3}$/i.test(input)) throw new Error('请输入三位币种代码')
    return input.toUpperCase()
  }
  return input
}
