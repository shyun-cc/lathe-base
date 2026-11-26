<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RequestError, request, setToken, token } from './api'
import { groups, resources } from './records'
import PrintPage from './PrintPage.vue'
import RecordsPage from './RecordsPage.vue'
import TemplatesPage from './TemplatesPage.vue'

const route = ref(location.hash)
const menuOpen = ref(false)
const credential = ref('')
const checking = ref(Boolean(token.value))
const loginError = ref('')
const errorText = ref('')
const errorDialog = ref<HTMLDialogElement>()
const feedback = ref('')
let feedbackTimer: ReturnType<typeof setTimeout> | undefined

const segments = computed(() => route.value.replace(/^#\/?/, '').split('/'))
const kind = computed(() => segments.value[0] || 'home')
const id = computed(() => /^[1-9]\d*$/.test(segments.value[1] ?? '') ? segments.value[1] : undefined)
const printParents: Record<string, string> = { 'sales-order': 'sales', 'work-order': 'production', shipment: 'shipping' }
const printParent = computed(() => Object.hasOwn(printParents, segments.value[1]) ? printParents[segments.value[1]] : undefined)
const isPrint = computed(() => kind.value === 'print' && Boolean(printParent.value) && /^[1-9]\d*$/.test(segments.value[2] ?? ''))
const active = computed(() => isPrint.value ? printParent.value : kind.value)
const activeGroup = computed(() => groups.find(group => group.items.includes(active.value ?? "")))
const label = (key: string) => resources[key]?.label ?? ({ home: '工作台', templates: '打印模板', print: '打印预览' }[key] || '页面未找到')

function onRoute() {
  route.value = location.hash
  menuOpen.value = false
  window.scrollTo(0, 0)
}

onMounted(async () => {
  window.addEventListener('hashchange', onRoute)
  if (!token.value) return
  try {
    await request('session')
  } catch (error) {
    setToken('')
    showError(error)
  } finally {
    checking.value = false
  }
})

onUnmounted(() => {
  window.removeEventListener('hashchange', onRoute)
  clearTimeout(feedbackTimer)
})

watch(token, value => {
  if (!value) menuOpen.value = false
})

async function login() {
  if (checking.value) return
  const value = credential.value.trim()
  loginError.value = ''
  if (!value) {
    loginError.value = '请输入实例访问令牌。'
    return
  }
  checking.value = true
  try {
    await request('session', undefined, 'GET', value)
    setToken(value)
    credential.value = ''
  } catch (error) {
    showError(error)
  } finally {
    checking.value = false
  }
}

async function showError(error: unknown) {
  errorText.value = error instanceof Error ? error.message : '操作失败。'
  if (error instanceof RequestError && error.status === 401) feedback.value = ''
  await nextTick()
  if (!errorDialog.value?.open) errorDialog.value?.showModal()
}

function showFeedback(message: string) {
  feedback.value = message
  clearTimeout(feedbackTimer)
  feedbackTimer = setTimeout(() => { feedback.value = '' }, 4500)
}
</script>

<template>
  <div v-if="!token || checking" class="login">
    <div class="login-panel">
      <div class="brand login-brand"><span class="brand-mark">母</span><span>母版<span class="brand-sub">Lathe Base</span></span></div>
      <h1>登录工厂工作台</h1>
      <p class="muted">使用管理员为此实例配置的访问令牌。</p>
      <form novalidate @submit.prevent="login">
        <label class="field">实例访问令牌
          <input v-model="credential" type="password" autocomplete="off" :aria-invalid="Boolean(loginError)" @input="loginError = ''">
        </label>
        <p v-if="loginError" class="error-text" role="alert">{{ loginError }}</p>
        <button class="button primary" :disabled="checking">{{ checking ? '正在验证…' : '进入工作台' }}</button>
      </form>
    </div>
  </div>
  <div v-else class="shell" :class="{ 'printing-route': isPrint }">
    <aside id="factory-nav" class="sidebar" :class="{ 'menu-open': menuOpen }">
      <a class="brand" href="#/"><span class="brand-mark">母</span><span>母版<span class="brand-sub">Lathe Base</span></span></a>
      <nav aria-label="工作台导航">
        <a href="#/" :aria-current="kind === 'home' ? 'page' : undefined">工作台</a>
        <div v-for="group in groups" :key="group.label" class="nav-group">
          <p>{{ group.label }}</p>
          <a v-for="key in group.items" :key="key" :href="`#/${key}`" :aria-current="active === key ? 'page' : undefined">{{ label(key) }}</a>
        </div>
      </nav>
    </aside>
    <div class="workspace">
      <header class="topbar">
        <div class="top-context">
          <button class="mobile-toggle button secondary" aria-controls="factory-nav" :aria-expanded="menuOpen" @click="menuOpen = !menuOpen">☰ 菜单</button>
          <span>{{ activeGroup?.label ?? '母版' }}</span>
          <span aria-hidden="true">/</span>
          <strong>{{ label(kind) }}</strong>
        </div>
        <button class="text-button logout" @click="setToken('')">退出登录</button>
      </header>
      <main id="main" class="content">
        <template v-if="kind === 'home'">
          <header class="welcome">
            <h1>工厂工作台</h1>
            <p>维护基础档案，跟进生产与交付，处理往来结算。</p>
            <a class="button primary" href="#/sales">进入销售订单 <span aria-hidden="true">→</span></a>
          </header>
          <div class="home-grid">
            <section v-for="group in groups" :key="group.label" class="surface shortcut">
              <h2>{{ group.label }}</h2>
              <div class="shortcut-links">
                <a v-for="key in group.items" :key="key" :href="`#/${key}`">{{ label(key) }} <span aria-hidden="true">↗</span></a>
              </div>
            </section>
          </div>
        </template>
        <RecordsPage v-else-if="Object.hasOwn(resources, kind)" :key="route" :kind="kind" :id="id" @error="showError" @feedback="showFeedback" />
        <TemplatesPage v-else-if="kind === 'templates'" @error="showError" @feedback="showFeedback" />
        <PrintPage v-else-if="isPrint" :key="route" :kind="segments[1]" :id="segments[2]" :back="`#/${printParent}/${segments[2]}`" @error="showError" />
        <div v-else class="surface empty">
          <h1>页面未找到</h1>
          <a class="text-button" href="#/">返回工作台</a>
        </div>
      </main>
    </div>
  </div>
  <p v-if="feedback && token" class="toast" role="status">{{ feedback }}</p>
  <dialog ref="errorDialog" class="modal compact" aria-labelledby="error-title" aria-describedby="error-message">
    <header class="modal-head"><h2 id="error-title">操作未完成</h2></header>
    <p id="error-message" class="modal-body">{{ errorText }}</p>
    <footer class="modal-foot"><button class="button primary" @click="errorDialog?.close()">知道了</button></footer>
  </dialog>
</template>
