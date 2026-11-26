<script setup lang="ts">
import type { PrintCell, PrintField } from './printing'

defineProps<{ title: string; fields: PrintField[]; span: number }>()
const cells = defineModel<PrintCell[]>({ required: true })

function move(index: number, offset: number) {
  const target = index + offset
  if (target < 0 || target >= cells.value.length) return
  const [cell] = cells.value.splice(index, 1)
  cells.value.splice(target, 0, cell)
}
</script>

<template>
  <fieldset class="template-section">
    <legend>{{ title }}</legend>
    <div v-for="(cell, index) in cells" :key="index" class="editor-row">
      <div class="fields">
        <label class="field">标签<input v-model="cell.label" maxlength="80"></label>
        <label class="field">内容来源
          <select v-model="cell.field" @change="cell.field && (cell.text = '')">
            <option value="">固定文字或签名栏</option>
            <option v-for="field in fields" :key="field.key" :value="field.key">{{ field.label }}</option>
          </select>
        </label>
        <label v-if="!cell.field" class="field">固定文字<input v-model="cell.text" maxlength="500"></label>
        <label class="field">占栏数<input v-model.number="cell.span" type="number" min="1" :max="span"></label>
      </div>
      <div class="actions">
        <button type="button" class="text-button" :disabled="index === 0" @click="move(index, -1)">上移</button>
        <button type="button" class="text-button" :disabled="index === cells.length - 1" @click="move(index, 1)">下移</button>
        <button type="button" class="text-button" @click="cells.splice(index, 1)">移除</button>
      </div>
    </div>
    <button type="button" class="button secondary" :disabled="cells.length >= 24"
      @click="cells.push({ label: '', field: '', text: '', span: 1 })">+ 添加字段</button>
  </fieldset>
</template>
