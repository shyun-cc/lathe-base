<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { all, page, pageSize, request } from './api'
import type { ID, Row } from './api'
import { actionForms, createForm, masterForm, shipmentForm, splitForm } from './forms'
import type { FormSpec } from './forms'
import { column, display, resources } from './records'
import type { Column } from './records'
import RecordForm from './RecordForm.vue'
import RecordsTable from './RecordsTable.vue'

const props = defineProps<{ kind: string; id?: string }>()
const emit = defineEmits<{ error: [error: unknown]; feedback: [message: string] }>()
const resource = resources[props.kind]
const rows = ref<Row[]>([])
const record = ref<Row>()
const references = ref<Record<string, Row[]>>({})
const cursors = ref(['0'])
const hasNext = ref(false)
const loading = ref(false)
const form = ref<FormSpec>()
const confirm = ref<{ title: string; path: string; method: string }>()
const confirmDialog = ref<HTMLDialogElement>()
const committing = ref(false)
const newRecord = resource.master ? masterForm(resource) : createForm(props.kind)

const extra: Record<string, Column[]> = {
  customers: [column('is_supplier', '同时为供应商')],
  suppliers: [column('is_customer', '同时为客户')],
  outsource: [column('currency', '币种'), column('amount_minor', '委外金额（最小单位）'), column('sent_on', '发出日期'), column('due_on', '预计回收日期')],
  shipping: [column('currency', '币种')],
  statements: [column('period_start', '账期起始'), column('period_end', '账期截止'), column('remaining_minor', '待核销（最小单位）')],
  cash: [column('currency', '币种'), column('method', '收付方式'), column('reference', '参考号'), column('remaining_minor', '未核销（最小单位）')],
  invoices: [column('currency', '币种'), column('tax_amount_minor', '税额（最小单位）'), column('statement_id', '关联对账单', 'statements'), column('notes', '备注')],
}
const detailColumns = [...resource.columns, ...(extra[props.kind] ?? [])]
const sections = computed(() => {
  const sales = [column('line_no', '行号'), column('product_name', '物料'), column('unit_id', '单位', 'units'), column('quantity_milli', '数量'), column('amount_minor', '金额（最小单位）'), column('due_on', '交期')]
  const source = record.value?.direction === 'payable' ? column('outsource_job_id', '委外单', 'outsource-jobs') : column('shipment_id', '发货单', 'shipments')
  const allocation = props.kind === 'cash' ? column('statement_id', '对账单', 'statements') : column('cash_entry_id', '收付流水', 'cash-entries')
  return {
    sales: [{ key: 'lines', label: '订单明细', columns: sales }],
    production: [{ key: 'reports', label: '报工记录', columns: [column('report_no', '报工编号'), column('operation_id', '工序', 'operations'), column('employee_id', '员工', 'employees'), column('quantity_milli', '数量'), column('reported_on', '报工日期')] }],
    outsource: [{ key: 'returns', label: '回收记录', columns: [column('return_no', '回收单号'), column('received_on', '回收日期'), column('accepted_milli', '合格数量'), column('rejected_milli', '不合格数量')] }],
    shipping: [{ key: 'lines', label: '发货明细', columns: [column('product_name', '物料'), column('quantity_milli', '发货数量'), column('amount_minor', '金额（最小单位）')] }],
    statements: [
      { key: 'lines', label: '对账来源', columns: [source, column('amount_minor', '金额（最小单位）')] },
      { key: 'allocations', label: '核销记录', columns: [allocation, column('amount_minor', '本次核销（最小单位）')] },
    ],
    cash: [{ key: 'allocations', label: '核销记录', columns: [allocation, column('amount_minor', '本次核销（最小单位）')] }],
  }[props.kind] ?? []
})

onMounted(load)

async function load() {
  loading.value = true
  try {
    if (props.id) {
      record.value = await request<Row>(`${resource.path}/${props.id}`)
    } else {
      const listPath = resource.role ? props.kind : resource.path
      const result = await page<Row>(listPath, cursors.value[cursors.value.length - 1], pageSize + 1)
      hasNext.value = result.length > pageSize
      rows.value = result.slice(0, pageSize)
    }
    const columns = props.id ? [...detailColumns, ...sections.value.flatMap(section => section.columns)] : resource.columns
    const paths = [...new Set(columns.flatMap(field => field.reference ?? []))]
    await Promise.all(paths.map(async path => {
      const data = props.id ? [record.value!] : rows.value
      const children = sections.value.flatMap(section => (record.value?.[section.key] as Row[] | undefined) ?? [])
      const ids = [...new Set(columns.filter(field => field.reference === path)
        .flatMap(field => [...data, ...children].map(row => row[field.key]))
        .filter((id): id is ID => typeof id === 'number' || typeof id === 'bigint'))]
      if (ids.length <= 6) references.value[path] = await Promise.all(ids.map(id => request<Row>(`${path}/${id}`)))
      else references.value[path] = await all<Row>(path)
    }))
  } catch (error) {
    emit('error', error)
  } finally {
    loading.value = false
  }
}

function advance(next: boolean) {
  if (next) cursors.value.push(String(rows.value[rows.value.length - 1].id))
  else cursors.value.pop()
  void load()
}

function saved(ids: ID[]) {
  const target = form.value?.result
  form.value = undefined
  emit('feedback', ids.length > 1 ? `已保存 ${ids.length} 条记录` : '已保存')
  if (target) location.hash = ids.length === 1 ? `#/${target}/${ids[0]}` : `#/${target}`
  else void load()
}

function ask(title: string, path: string, method = 'POST') {
  confirm.value = { title, path, method }
  confirmDialog.value?.showModal()
}

async function confirmed() {
  if (!confirm.value || committing.value) return
  const deleting = confirm.value.method === 'DELETE'
  committing.value = true
  try {
    await request(confirm.value.path, undefined, confirm.value.method)
    confirmDialog.value?.close()
    confirm.value = undefined
    emit('feedback', deleting ? '已删除' : '操作完成')
    if (deleting && props.id) location.hash = `#/${props.kind}`
    else await load()
  } catch (error) {
    emit('error', error)
  } finally {
    committing.value = false
  }
}
</script>

<template>
  <template v-if="!id">
    <header class="page-head">
      <h1>{{ resource.label }}</h1>
      <div class="actions">
        <button class="button secondary" :disabled="loading" @click="load">刷新</button>
        <button v-if="newRecord" class="button primary" @click="form = newRecord">+ {{ newRecord.title }}</button>
        <a v-else-if="kind === 'shipping'" class="button primary" href="#/sales">从销售订单创建</a>
      </div>
    </header>
    <p v-if="kind === 'production'" class="page-note">从已确认的销售订单明细拆分工单。</p>
    <section class="surface" :aria-busy="loading">
      <p v-if="loading && !rows.length" class="empty" role="status">正在读取记录…</p>
      <RecordsTable v-else :columns="resource.columns" :rows="rows" :references="references" :caption="resource.label" action="打开">
        <template #default="{ row }">
          <a class="text-button" :href="`#/${kind}/${row.id}`">查看</a>
          <button v-if="resource.master" class="text-button" @click="form = masterForm(resource, row)">编辑</button>
        </template>
      </RecordsTable>
      <footer class="pagination">
        <span class="muted">第 {{ cursors.length }} 页 · {{ rows.length }} 条</span>
        <div class="actions">
          <button class="button secondary" :disabled="loading || cursors.length === 1" @click="advance(false)">上一页</button>
          <button class="button secondary" :disabled="loading || !hasNext" @click="advance(true)">下一页</button>
        </div>
      </footer>
    </section>
  </template>
  <template v-else>
    <a class="back-link" :href="`#/${kind}`">← {{ resource.label }}</a>
    <template v-if="record">
      <header class="page-head">
        <div>
          <p class="section-label">{{ resource.label }}</p>
          <h1>{{ record[resource.name] }}</h1>
        </div>
        <div class="actions">
          <template v-if="resource.master">
            <button class="button secondary" @click="ask('删除这条档案？', `${resource.path}/${id}`, 'DELETE')">删除</button>
            <button class="button primary" @click="form = masterForm(resource, record)">编辑</button>
          </template>
          <a v-if="resource.printType" class="button secondary" :href="`#/print/${resource.printType}/${id}`">打印预览</a>
          <button v-if="kind === 'sales' && record.status === 'draft'" class="button primary"
            @click="ask('确认这张销售订单？', `${resource.path}/${id}/confirm`)">确认订单</button>
          <button v-if="kind === 'sales' && record.status === 'confirmed'" class="button primary" @click="form = shipmentForm(record)">新建发货单</button>
          <button v-if="kind === 'invoices' && record.status !== 'void'" class="button secondary"
            @click="ask('作废这张发票登记？', `${resource.path}/${id}/void`)">作废</button>
          <button v-for="operation in actionForms(kind, record)" :key="operation.path" class="button primary" @click="form = operation">{{ operation.title }}</button>
        </div>
      </header>
      <dl class="record-facts surface">
        <div v-for="field in detailColumns" :key="field.key">
          <dt>{{ field.label }}</dt>
          <dd v-if="field.key === 'is_supplier' || field.key === 'is_customer'">{{ record[field.key] ? '是' : '否' }}</dd>
          <dd v-else>{{ display(record, field, references) }}</dd>
        </div>
      </dl>
      <section v-for="section in sections" :key="section.key" class="surface detail-section">
        <h2>{{ section.label }}</h2>
        <RecordsTable :columns="section.columns" :rows="(record[section.key] as Row[]) ?? []" :references="references"
          :caption="section.label" :action="kind === 'sales' && record.status === 'confirmed' ? '拆分' : undefined">
          <template #default="{ row }"><button class="text-button" @click="form = splitForm(row)">拆工单</button></template>
        </RecordsTable>
      </section>
    </template>
    <p v-else class="surface empty">{{ loading ? '正在读取单据…' : '无法读取此单据' }}</p>
  </template>
  <RecordForm v-if="form" :spec="form" @close="form = undefined" @saved="saved" @error="emit('error', $event)" />
  <dialog ref="confirmDialog" class="modal compact" aria-labelledby="confirm-title" @cancel.prevent="!committing && confirmDialog?.close()">
    <header class="modal-head"><h2 id="confirm-title">{{ confirm?.title }}</h2></header>
    <footer class="modal-foot">
      <button class="button secondary" :disabled="committing" @click="confirmDialog?.close()">取消</button>
      <button class="button primary" :disabled="committing" @click="confirmed">{{ committing ? '处理中…' : '确定' }}</button>
    </footer>
  </dialog>
</template>
