<script setup lang="ts">
import ShwStatusTag from './vendor/shw-wui/StatusTag.vue'
import type { Row } from './api'
import { display, statuses } from './records'
import type { Column } from './records'

defineProps<{
  columns: Column[]
  rows: Row[]
  references: Record<string, Row[]>
  action?: string
  caption: string
}>()

const statusOptions = [
  ...Object.entries(statuses).map(([value, label]) => ({
    value, label, type: ['cancelled', 'void'].includes(value) ? 'danger' as const
      : ['draft', 'planned'].includes(value) ? 'info' as const
      : ['released', 'in_progress', 'sent'].includes(value) ? 'warning' as const : 'success' as const,
  })),
  { value: 'true', label: '启用', type: 'success' as const },
  { value: 'false', label: '停用', type: 'info' as const },
]
</script>

<template>
  <div class="table-wrap">
    <table class="record-table">
      <caption class="sr-only">{{ caption }}</caption>
      <thead>
        <tr>
          <th v-for="field in columns" :key="field.key" scope="col">{{ field.label }}</th>
          <th v-if="action" class="table-action" scope="col">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in rows" :key="String(row.id ?? row.sales_order_line_id ?? index)">
          <td v-for="field in columns" :key="field.key" :data-label="field.label">
            <ShwStatusTag v-if="field.key === 'status' || field.key === 'active'"
              :value="String(row[field.key])" :options="statusOptions" :fallback-label="display(row, field, references)" size="small" />
            <span v-else>{{ display(row, field, references) }}</span>
          </td>
          <td v-if="action" class="table-action"><slot :row="row" /></td>
        </tr>
      </tbody>
    </table>
    <p v-if="!rows.length" class="empty">暂无记录</p>
  </div>
</template>
