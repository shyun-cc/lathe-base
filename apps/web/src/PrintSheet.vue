<script setup lang="ts">
import { computed } from 'vue'
import type { Configuration, PrintDocument } from './printing'

const props = defineProps<{ config: Configuration; data: PrintDocument }>()
const grid = computed(() => ({ gridTemplateColumns: `repeat(${props.config.columnsPerRow}, minmax(0, 1fr))` }))
const widths = computed(() => {
  const sum = props.config.columns.reduce((total, column) => total + column.width, 0)
  return props.config.columns.map(column => `${column.width / sum * 100}%`)
})
</script>

<template>
  <article class="print-sheet" :class="[config.paper.toLowerCase(), config.orientation]">
    <h1>{{ config.title }}</h1>
    <dl v-if="config.header.length" class="print-grid" :style="grid">
      <div v-for="(cell, index) in config.header" :key="index" :style="{ gridColumn: `span ${cell.span}` }">
        <dt v-if="cell.label">{{ cell.label }}</dt>
        <dd>{{ cell.field ? data.fields[cell.field] ?? '—' : cell.text || '____________' }}</dd>
      </div>
    </dl>
    <table class="print-table">
      <colgroup><col v-for="(width, index) in widths" :key="index" :style="{ width }"></colgroup>
      <thead><tr><th v-for="(column, index) in config.columns" :key="index" scope="col" :style="{ textAlign: column.align }">{{ column.label }}</th></tr></thead>
      <tbody>
        <tr v-for="(row, index) in data.rows" :key="index">
          <td v-for="(column, col) in config.columns" :key="col" :style="{ textAlign: column.align }">{{ row[column.field] ?? '—' }}</td>
        </tr>
      </tbody>
    </table>
    <dl v-if="config.footer.length" class="print-grid print-footer" :style="grid">
      <div v-for="(cell, index) in config.footer" :key="index" :style="{ gridColumn: `span ${cell.span}` }">
        <dt v-if="cell.label">{{ cell.label }}</dt>
        <dd>{{ cell.field ? data.fields[cell.field] ?? '—' : cell.text || '____________' }}</dd>
      </div>
    </dl>
  </article>
</template>
