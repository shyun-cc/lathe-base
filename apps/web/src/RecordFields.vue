<script setup lang="ts">
import type { Row } from './api'
import { sourceOf } from './forms'
import type { Field, Values } from './forms'
import { describe } from './records'

const props = defineProps<{
  fields: Field[]
  values: Values
  errors: Record<string, string>
  references: Record<string, Row[]>
  prefix: string
}>()
const emit = defineEmits<{ change: [key: string, value: string | boolean] }>()

function options(field: Field) {
  if (field.options) return field.options
  const source = sourceOf(field, props.values)
  if (!source) return []
  return (props.references[source] ?? []).filter(row => !field.filter || field.filter(row, props.values))
    .map(row => ({ value: String(row.id), label: describe(row, source) }))
}

function changeMultiple(field: Field, value: string, checked: boolean) {
  const selected = new Set(String(props.values[field.key]).split(',').filter(Boolean))
  if (checked) selected.add(value)
  else selected.delete(value)
  emit('change', field.key, [...selected].join(','))
}
</script>

<template>
  <div class="fields">
    <div v-for="field in fields" :key="field.key" class="field" :class="{ wide: field.type === 'multiple' }">
      <label v-if="field.type === 'boolean'" class="check">
        <input type="checkbox" :checked="Boolean(values[field.key])" @change="emit('change', field.key, ($event.target as HTMLInputElement).checked)">
        {{ field.label }}
      </label>
      <template v-else>
        <label v-if="field.type !== 'multiple'" :for="prefix + field.key">
          {{ field.label }} <span v-if="field.optional" class="muted">选填</span>
        </label>
        <fieldset v-if="field.type === 'multiple'" class="source-options" :aria-invalid="Boolean(errors[field.key])">
          <legend>{{ field.label }}</legend>
          <p v-if="!options(field).length" class="muted">当前条件下暂无可选来源</p>
          <label v-for="item in options(field)" :key="item.value" class="check">
            <input type="checkbox" :checked="String(values[field.key]).split(',').includes(item.value)"
              @change="changeMultiple(field, item.value, ($event.target as HTMLInputElement).checked)">
            {{ item.label }}
          </label>
        </fieldset>
        <select v-else-if="field.type === 'select'" :id="prefix + field.key" :value="values[field.key]"
          :aria-invalid="Boolean(errors[field.key])" @change="emit('change', field.key, ($event.target as HTMLSelectElement).value)">
          <option value="">{{ field.optional ? '不关联' : '请选择' }}</option>
          <option v-for="item in options(field)" :key="item.value" :value="item.value">{{ item.label }}</option>
        </select>
        <input v-else :id="prefix + field.key" :type="field.type === 'date' ? 'date' : 'text'"
          :inputmode="field.type === 'quantity' ? 'decimal' : field.type === 'money' ? 'numeric' : undefined"
          :value="values[field.key]" :aria-invalid="Boolean(errors[field.key])"
          @input="emit('change', field.key, ($event.target as HTMLInputElement).value)">
      </template>
      <small v-if="errors[field.key]" class="error-text">{{ errors[field.key] }}</small>
    </div>
  </div>
</template>
