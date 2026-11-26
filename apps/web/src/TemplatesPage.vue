<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { all, request } from './api'
import type { ID } from './api'
import { parseConfiguration, sample } from './printing'
import type { Configuration, Definition, Template } from './printing'
import PrintSheet from './PrintSheet.vue'
import TemplateCells from './TemplateCells.vue'

const emit = defineEmits<{ error: [error: unknown]; feedback: [message: string] }>()
const definitions = ref<Record<string, Definition>>({})
const templates = ref<Template[]>([])
const kind = ref('')
const version = ref('')
const code = ref('')
const name = ref('')
const configuration = ref<Configuration>()
const errorText = ref('')
const jsonError = ref('')
const jsonDraft = ref('')
const jsonOpen = ref(false)
const busy = ref(false)
const snapshot = ref('')
const definition = computed(() => definitions.value[kind.value])
const versions = computed(() => templates.value.filter(template => template.document_type === kind.value).slice().reverse())
const selected = computed(() => versions.value.find(template => String(template.id) === version.value))
const current = computed(() => JSON.stringify([code.value, name.value, configuration.value]))
const dirty = computed(() => current.value !== snapshot.value)
const publishDialog = ref<HTMLDialogElement>()

onMounted(async () => {
  try {
    [definitions.value, templates.value] = await Promise.all([
      request<Record<string, Definition>>('print-definitions'), all<Template>('print-templates'),
    ])
    kind.value = definitions.value['sales-order'] ? 'sales-order' : Object.keys(definitions.value)[0]
    loadVersion('')
  } catch (error) {
    emit('error', error)
  }
})

function loadVersion(id: string) {
  version.value = id
  const template = selected.value
  code.value = template?.code ?? ''
  name.value = template?.name ?? ''
  configuration.value = JSON.parse(JSON.stringify(template?.configuration ?? definition.value.default)) as Configuration
  jsonOpen.value = false
  snapshot.value = current.value
  errorText.value = ''
  jsonError.value = ''
}

function toggleJSON() {
  if (!jsonOpen.value) jsonDraft.value = JSON.stringify(configuration.value, null, 2)
  jsonOpen.value = !jsonOpen.value
  jsonError.value = ''
}

function applyJSON() {
  try {
    configuration.value = parseConfiguration(JSON.parse(jsonDraft.value), definition.value)
    jsonError.value = ''
    jsonOpen.value = false
  } catch (error) {
    jsonError.value = (error as Error).message
  }
}

function moveColumn(index: number, offset: number) {
  if (!configuration.value) return
  const columns = configuration.value.columns
  const target = index + offset
  if (target < 0 || target >= columns.length) return
  const [column] = columns.splice(index, 1)
  columns.splice(target, 0, column)
}

function addColumn() {
  const field = definition.value.detail[0]
  configuration.value?.columns.push({ field: field.key, label: field.label, width: 1, align: 'left' })
}

async function save() {
  if (busy.value) return
  errorText.value = ''
  let payload: Configuration
  try {
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,39}$/.test(code.value)) throw new Error('编号须使用 1–40 位字母、数字、下划线或连字符，并以字母或数字开头。')
    if (!name.value.trim()) throw new Error('请输入模板名称。')
    payload = parseConfiguration(configuration.value, definition.value)
  } catch (error) {
    errorText.value = (error as Error).message
    return
  }
  busy.value = true
  try {
    const result = await request<{ id: ID }>('print-templates', { document_type: kind.value, code: code.value, name: name.value, configuration: payload })
    templates.value = await all<Template>('print-templates')
    loadVersion(String(result.id))
    emit('feedback', '模板版本已保存，可以发布。')
  } catch (error) {
    emit('error', error)
  } finally {
    busy.value = false
  }
}

async function publish() {
  if (!selected.value || busy.value) return
  busy.value = true
  try {
    await request(`print-templates/${selected.value.id}/publish`, undefined, 'POST')
    templates.value = await all<Template>('print-templates')
    publishDialog.value?.close()
    emit('feedback', '模板版本已发布。')
  } catch (error) {
    emit('error', error)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <header class="page-head">
    <h1>打印模板</h1>
  </header>
  <p v-if="!configuration" class="surface empty">正在读取打印模板…</p>
  <div v-else class="template-layout">
    <form class="surface template-form" novalidate @submit.prevent="save">
      <div class="fields">
        <label class="field">单据类别
          <select :value="kind" :disabled="busy" @change="kind = ($event.target as HTMLSelectElement).value; loadVersion('')">
            <option v-for="(item, key) in definitions" :key="key" :value="key">{{ item.label }}</option>
          </select>
        </label>
        <label class="field">模板版本
          <select :value="version" :disabled="busy" @change="loadVersion(($event.target as HTMLSelectElement).value)">
            <option value="">内置模板</option>
            <option v-for="item in versions" :key="String(item.id)" :value="String(item.id)">
              {{ item.name }} · {{ item.code }} v{{ item.version }}{{ item.active ? '（生效）' : '' }}
            </option>
          </select>
        </label>
      </div>
      <p class="muted">保存时创建新版本。发布后，可在对应单据的打印预览中选择该模板。</p>
      <fieldset :disabled="busy" class="editor">
        <div class="fields">
          <label class="field">模板编号<input v-model.trim="code" maxlength="40" placeholder="例如 delivery"></label>
          <label class="field">模板名称<input v-model.trim="name" maxlength="80"></label>
          <label class="field wide">打印标题<input v-model="configuration.title" maxlength="80"></label>
          <label class="field">纸张<select v-model="configuration.paper"><option>A4</option><option>A5</option></select></label>
          <label class="field">方向
            <select v-model="configuration.orientation"><option value="portrait">纵向</option><option value="landscape">横向</option></select>
          </label>
          <label class="field">每行栏数
            <select v-model.number="configuration.columnsPerRow"
              @change="[...configuration.header, ...configuration.footer].forEach(cell => { cell.span = Math.min(cell.span, configuration!.columnsPerRow) })">
              <option v-for="count in 4" :key="count" :value="count">{{ count }}</option>
            </select>
          </label>
        </div>
        <TemplateCells v-model="configuration.header" title="表头" :fields="definition.fields" :span="configuration.columnsPerRow" />
        <fieldset class="template-section">
          <legend>明细列</legend>
          <div v-for="(item, index) in configuration.columns" :key="index" class="editor-row">
            <div class="fields">
              <label class="field">字段<select v-model="item.field">
                <option v-for="field in definition.detail" :key="field.key" :value="field.key">{{ field.label }}</option>
              </select></label>
              <label class="field">列名<input v-model="item.label" maxlength="80"></label>
              <label class="field">相对宽度<input v-model.number="item.width" type="number" min="1" max="100"></label>
              <label class="field">对齐<select v-model="item.align">
                <option value="left">左</option><option value="center">居中</option><option value="right">右</option>
              </select></label>
            </div>
            <div class="actions">
              <button type="button" class="text-button" :disabled="index === 0" @click="moveColumn(index, -1)">上移</button>
              <button type="button" class="text-button" :disabled="index === configuration.columns.length - 1" @click="moveColumn(index, 1)">下移</button>
              <button type="button" class="text-button" :disabled="configuration.columns.length === 1" @click="configuration.columns.splice(index, 1)">移除</button>
            </div>
          </div>
          <button type="button" class="button secondary" :disabled="configuration.columns.length >= 12" @click="addColumn">+ 添加明细列</button>
        </fieldset>
        <TemplateCells v-model="configuration.footer" title="表底栏" :fields="definition.fields" :span="configuration.columnsPerRow" />
        <button type="button" class="button secondary" :aria-expanded="jsonOpen" @click="toggleJSON">{{ jsonOpen ? '关闭 JSON' : '编辑 JSON' }}</button>
        <div v-if="jsonOpen" class="json-box">
          <label class="field">模板 JSON<textarea v-model="jsonDraft" rows="13" spellcheck="false" /></label>
          <p v-if="jsonError" class="error-text" role="alert">{{ jsonError }}</p>
          <button type="button" class="button secondary" @click="applyJSON">应用到表单</button>
        </div>
      </fieldset>
      <p v-if="errorText" class="error-text" role="alert">{{ errorText }}</p>
      <footer class="template-actions">
        <span v-if="dirty" class="muted">有未保存的更改</span>
        <div class="actions">
          <button class="button primary" :disabled="busy || jsonOpen">保存新版本</button>
          <button v-if="selected && !selected.active" type="button" class="button secondary" :disabled="busy || dirty"
            @click="publishDialog?.showModal()">发布此版本</button>
        </div>
      </footer>
    </form>
    <section class="template-preview">
      <h2>样例预览</h2>
      <PrintSheet :config="configuration" :data="sample(definition)" />
    </section>
  </div>
  <dialog ref="publishDialog" class="modal compact" aria-labelledby="publish-title" @cancel.prevent="!busy && publishDialog?.close()">
    <header class="modal-head"><h2 id="publish-title">发布此模板版本？</h2></header>
    <p class="modal-body">对应编号将使用 v{{ selected?.version }} 作为生效版本，历史版本仍可重新发布。</p>
    <footer class="modal-foot">
      <button class="button secondary" :disabled="busy" @click="publishDialog?.close()">取消</button>
      <button class="button primary" :disabled="busy" @click="publish">发布</button>
    </footer>
  </dialog>
</template>
