<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watchEffect } from 'vue'
import { all, request } from './api'
import { parseConfiguration } from './printing'
import type { Definition, PrintDocument, Template } from './printing'
import PrintSheet from './PrintSheet.vue'

const props = defineProps<{ kind: string; id: string; back: string }>()
const emit = defineEmits<{ error: [error: unknown] }>()
const definition = ref<Definition>()
const data = ref<PrintDocument>()
const templates = ref<Template[]>([])
const selection = ref('')
const failed = ref(false)
const configuration = computed(() => templates.value.find(template => String(template.id) === selection.value)?.configuration ?? definition.value?.default)
const stylesheet = document.createElement('style')

onMounted(async () => {
  document.head.append(stylesheet)
  try {
    const [definitions, documentData, saved] = await Promise.all([
      request<Record<string, Definition>>('print-definitions'),
      request<PrintDocument>(`print-documents/${props.kind}/${props.id}`),
      all<Template>('print-templates'),
    ])
    definition.value = definitions[props.kind]
    const type = definition.value
    data.value = documentData
    templates.value = saved.filter(template => template.active && template.document_type === props.kind).map(template =>
      ({ ...template, configuration: parseConfiguration(template.configuration, type) }))
  } catch (error) {
    failed.value = true
    emit('error', error)
  }
})

watchEffect(() => {
  const config = configuration.value
  if (config) stylesheet.textContent = `@page { size: ${config.paper} ${config.orientation}; margin: 11mm; }`
})

onBeforeUnmount(() => stylesheet.remove())

function print() {
  window.print()
}
</script>

<template>
  <div class="print-controls">
    <a class="back-link" :href="back">← 返回单据</a>
    <header class="page-head">
      <h1>打印预览</h1>
      <div v-if="configuration && !failed" class="actions">
        <label class="print-select">模板
          <select v-model="selection">
            <option value="">内置模板</option>
            <option v-for="template in templates" :key="String(template.id)" :value="String(template.id)">{{ template.name }} · v{{ template.version }}</option>
          </select>
        </label>
        <button class="button primary" @click="print">打印 / 保存 PDF</button>
      </div>
    </header>
  </div>
  <PrintSheet v-if="configuration && data && !failed" :config="configuration" :data="data" />
  <p v-else class="surface empty">{{ failed ? '无法生成打印预览' : '正在读取打印数据…' }}</p>
</template>
