<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { all, request } from './api'
import type { ID, Row } from './api'
import RecordFields from './RecordFields.vue'
import { defaults, encode, sourceOf } from './forms'
import type { Field, FormSpec, Values } from './forms'

const props = defineProps<{ spec: FormSpec }>()
const emit = defineEmits<{ close: []; saved: [ids: ID[]]; error: [error: unknown] }>()
const dialog = ref<HTMLDialogElement>()
const values = ref(defaults(props.spec.fields))
const errors = ref<Record<string, string>>({})
const lines = ref<{ key: number; values: Values; errors: Record<string, string> }[]>([])
const references = ref<Record<string, Row[]>>({})
const pending = ref(0)
const loading = computed(() => pending.value > 0)
const failedSources = ref<string[]>([])
const loadingPaths = new Set<string>()
const busy = ref(false)
const requestKey = props.spec.path === 'allocations' ? crypto.randomUUID() : undefined
let nextKey = 0

function addLine() {
  if (props.spec.lines) lines.value.push({ key: nextKey++, values: defaults(props.spec.lines.fields), errors: {} })
}

async function loadReferences(fields: Field[], input: Values) {
  const paths = [...new Set(fields.map(field => sourceOf(field, input)).filter((path): path is string => Boolean(path)))]
  const missing = paths.filter(path => !(path in references.value) && !loadingPaths.has(path))
  await Promise.all(missing.map(async path => {
    loadingPaths.add(path)
    pending.value++
    try {
      references.value[path] = await all<Row>(path)
      failedSources.value = failedSources.value.filter(value => value !== path)
    } catch (error) {
      if (!failedSources.value.includes(path)) failedSources.value.push(path)
      emit('error', error)
    } finally {
      loadingPaths.delete(path)
      pending.value--
    }
  }))
}

function loadOptions() {
  return loadReferences([...props.spec.fields, ...(props.spec.lines?.fields ?? [])], values.value)
}

onMounted(() => {
  dialog.value?.showModal()
  addLine()
  void loadOptions()
})

function change(key: string, value: string | boolean) {
  const previous = { ...values.value }
  values.value[key] = value
  delete errors.value[key]
  let changedSource = false
  for (const field of props.spec.fields) {
    const source = sourceOf(field, values.value)
    if (source !== sourceOf(field, previous)) {
      values.value[field.key] = ''
      changedSource = true
      continue
    }
    const chosen = values.value[field.key]
    if (key === field.key || !source || !chosen || !field.filter) continue
    const available = (references.value[source] ?? []).filter(row => field.filter?.(row, values.value))
    values.value[field.key] = String(chosen).split(',').filter(id => available.some(row => String(row.id) === id)).join(',')
  }
  if (changedSource) void loadOptions()
}

function toPayload(fields: Field[], input: Values, invalid: Record<string, string>) {
  const data: Record<string, unknown> = {}
  for (const field of fields) {
    delete invalid[field.key]
    try {
      data[field.key] = encode(field, input[field.key])
    } catch (error) {
      invalid[field.key] = (error as Error).message
    }
  }
  return data
}

async function submit() {
  if (busy.value || loading.value) return
  const body = { ...props.spec.fixed, ...toPayload(props.spec.fields, values.value, errors.value) }
  if (props.spec.lines) {
    const { key, fields } = props.spec.lines
    body[key] = lines.value.map(line => toPayload(fields, line.values, line.errors))
  }
  if (Object.keys(errors.value).length || lines.value.some(line => Object.keys(line.errors).length)) return
  if (requestKey) body.request_key = requestKey
  busy.value = true
  try {
    const result = await request<{ id: ID } | { id: ID }[]>(props.spec.path, body, props.spec.method)
    emit('saved', (Array.isArray(result) ? result : [result]).map(row => row.id))
  } catch (error) {
    emit('error', error)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <dialog ref="dialog" class="modal" @cancel.prevent="!busy && emit('close')" aria-labelledby="form-title">
    <form novalidate @submit.prevent="submit">
      <header class="modal-head">
        <h2 id="form-title">{{ spec.title }}</h2>
        <button type="button" class="icon-button" aria-label="关闭" :disabled="busy" @click="emit('close')">×</button>
      </header>
      <div class="modal-body">
        <p v-if="loading" role="status" class="muted">正在读取可选记录…</p>
        <button v-if="failedSources.length && !loading" type="button" class="button secondary" @click="loadOptions">重试读取可选记录</button>
        <RecordFields :fields="spec.fields" :values="values" :errors="errors" :references="references" prefix="field-" @change="change" />
        <section v-if="spec.lines" class="form-lines">
          <div class="split">
            <h3>{{ spec.lines.label }}</h3>
            <button type="button" class="button secondary" @click="addLine">+ 添加一行</button>
          </div>
          <div v-for="(line, index) in lines" :key="line.key" class="line">
            <div class="split">
              <strong>第 {{ index + 1 }} 行</strong>
              <button type="button" class="text-button" :disabled="lines.length === 1" @click="lines.splice(index, 1)">移除</button>
            </div>
            <RecordFields :fields="spec.lines.fields" :values="line.values" :errors="line.errors" :references="references" :prefix="`line-${line.key}-`"
              @change="(key, value) => { line.values[key] = value; delete line.errors[key] }" />
          </div>
        </section>
      </div>
      <footer class="modal-foot">
        <button type="button" class="button secondary" :disabled="busy" @click="emit('close')">取消</button>
        <button class="button primary" :disabled="busy || loading">{{ busy ? '提交中…' : '保存' }}</button>
      </footer>
    </form>
  </dialog>
</template>
