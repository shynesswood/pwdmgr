<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import {
  GetAppConfig,
  ReloadConfig,
  GetRepoStatus,
  Pull,
  Push,
  Sync,
  BindRepo,
  InitLocalVault,
  ListVaultEntries,
  AddVaultEntry,
  UpdateVaultEntry,
  DeleteVaultEntry,
} from '../wailsjs/go/app/App'
import { WindowSetBackgroundColour } from '../wailsjs/runtime/runtime'

function applyWindowChrome() {
  try {
    const dark = window.matchMedia('(prefers-color-scheme: dark)').matches
    if (dark) {
      WindowSetBackgroundColour(24, 24, 27, 255)
    } else {
      WindowSetBackgroundColour(250, 250, 252, 255)
    }
  } catch {
    /* non-Wails env */
  }
}

const activeTab = ref('vault')
const searchQuery = ref('')
const selectedTags = ref([])

const appConfig = ref({
  config_path: '',
  repo_root: '',
  remote_url: '',
  vault_file_name: 'vault.dat',
  load_error: '',
})

const status = ref(null)
const lastError = ref('')
const toastVisible = ref(false)
const toastMsg = ref('')
const toastType = ref('error')
let toastTimer = null

const vaultLabel = computed(() => appConfig.value.vault_file_name || 'vault.dat')

const vaultUnlocked = ref(false)
const sessionPassword = ref('')
const entries = ref([])
const formMode = ref('add')
const formOpen = ref(false)
const form = ref({
  id: '',
  name: '',
  username: '',
  password: '',
  note: '',
  tagsStr: '',
})
const pwdVisible = ref({})

const allTags = computed(() => {
  const set = new Set()
  for (const e of entries.value) {
    for (const t of e.tags || []) set.add(t)
  }
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-Hans'))
})

const toggleTag = (tag) => {
  const idx = selectedTags.value.indexOf(tag)
  if (idx === -1) {
    selectedTags.value = [...selectedTags.value, tag]
  } else {
    selectedTags.value = selectedTags.value.filter((t) => t !== tag)
  }
}

const clearTagFilter = () => {
  selectedTags.value = []
}

const filteredEntries = computed(() => {
  let list = entries.value
  if (selectedTags.value.length > 0) {
    list = list.filter((e) =>
      selectedTags.value.every((st) => (e.tags || []).includes(st)),
    )
  }
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(
      (e) =>
        (e.name || '').toLowerCase().includes(q) ||
        (e.username || '').toLowerCase().includes(q) ||
        (e.tags || []).some((t) => t.toLowerCase().includes(q)),
    )
  }
  return list
})

const dlgPwdOpen = ref(false)
const dlgPwdTitle = ref('')
const dlgPwdValue = ref('')
const dlgPwdInputRef = ref(null)
let dlgPwdResolve = null

const askPasswordAsync = (title) => {
  return new Promise((resolve) => {
    dlgPwdTitle.value = title
    dlgPwdValue.value = ''
    dlgPwdResolve = resolve
    dlgPwdOpen.value = true
    nextTick(() => {
      dlgPwdInputRef.value?.focus?.()
    })
  })
}

const dlgPwdConfirm = () => {
  const v = dlgPwdValue.value.trim()
  dlgPwdOpen.value = false
  const r = dlgPwdResolve
  dlgPwdResolve = null
  if (r) r(v === '' ? null : v)
}

const dlgPwdCancel = () => {
  dlgPwdOpen.value = false
  const r = dlgPwdResolve
  dlgPwdResolve = null
  if (r) r(null)
}

const dlgConfirmOpen = ref(false)
const dlgConfirmMsg = ref('')
let dlgConfirmResolve = null

const askConfirmAsync = (message) => {
  return new Promise((resolve) => {
    dlgConfirmMsg.value = message
    dlgConfirmResolve = resolve
    dlgConfirmOpen.value = true
  })
}

const dlgConfirmYes = () => {
  dlgConfirmOpen.value = false
  const r = dlgConfirmResolve
  dlgConfirmResolve = null
  if (r) r(true)
}

const dlgConfirmNo = () => {
  dlgConfirmOpen.value = false
  const r = dlgConfirmResolve
  dlgConfirmResolve = null
  if (r) r(false)
}

const refreshAppConfig = async () => {
  appConfig.value = await GetAppConfig()
}

const parseTags = (s) =>
  String(s || '')
    .split(/[,，]/)
    .map((t) => t.trim())
    .filter(Boolean)

const resetVaultForm = () => {
  formMode.value = 'add'
  form.value = { id: '', name: '', username: '', password: '', note: '', tagsStr: '' }
}

const lockVault = () => {
  vaultUnlocked.value = false
  sessionPassword.value = ''
  entries.value = []
  pwdVisible.value = {}
  selectedTags.value = []
  resetVaultForm()
  formOpen.value = false
}

const togglePwdRow = (id) => {
  pwdVisible.value = { ...pwdVisible.value, [id]: !pwdVisible.value[id] }
}

const showToast = (msg, type = 'error') => {
  toastMsg.value = msg
  toastType.value = type
  toastVisible.value = true
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastVisible.value = false
  }, 4000)
}

const clearError = () => {
  lastError.value = ''
}

const showErr = (e) => {
  const msg = typeof e === 'string' ? e : e?.message ?? String(e)
  lastError.value = msg
  showToast(msg, 'error')
}

const showSuccess = (msg) => {
  showToast(msg, 'success')
}

const doUnlockVault = async () => {
  clearError()
  const p = await askPasswordAsync(`输入主密码以打开加密库（${vaultLabel.value}）`)
  if (p == null) return
  try {
    entries.value = await ListVaultEntries(p)
    sessionPassword.value = p
    vaultUnlocked.value = true
    resetVaultForm()
    showSuccess('保险库已解锁')
  } catch (e) {
    showErr(e)
  }
}

const refreshVaultEntries = async () => {
  if (!vaultUnlocked.value || !sessionPassword.value) return
  clearError()
  try {
    entries.value = await ListVaultEntries(sessionPassword.value)
    showSuccess('条目已刷新')
  } catch (e) {
    showErr(e)
    lockVault()
  }
}

const startEditEntry = (row) => {
  formMode.value = 'edit'
  form.value = {
    id: row.id,
    name: row.name || '',
    username: row.username || '',
    password: row.password || '',
    note: row.note || '',
    tagsStr: (row.tags || []).join(', '),
  }
  formOpen.value = true
}

const submitVaultForm = async () => {
  clearError()
  const p = sessionPassword.value
  if (!p) {
    showErr('请先解锁保险库并输入主密码')
    return
  }
  const { id, name, username, password, note, tagsStr } = form.value
  if (!String(name).trim()) {
    showErr('请填写条目名称')
    return
  }
  const tags = parseTags(tagsStr)
  const tagList = Array.isArray(tags) ? tags : []
  try {
    if (formMode.value === 'add') {
      await AddVaultEntry(p, name, username, password, note, tagList)
      showSuccess('条目已添加')
    } else {
      await UpdateVaultEntry(p, { id, name, username, password, note, tags, updated_at: 0 })
      showSuccess('条目已更新')
    }
    entries.value = await ListVaultEntries(p)
    resetVaultForm()
    formOpen.value = false
  } catch (e) {
    showErr(e)
  }
}

const deleteEntry = async (row) => {
  const ok = await askConfirmAsync(`确定删除「${row.name || row.id}」？此操作不可撤销。`)
  if (!ok) return
  clearError()
  try {
    await DeleteVaultEntry(sessionPassword.value, row.id)
    entries.value = await ListVaultEntries(sessionPassword.value)
    if (formMode.value === 'edit' && form.value.id === row.id) {
      resetVaultForm()
      formOpen.value = false
    }
    showSuccess('条目已删除')
  } catch (e) {
    showErr(e)
  }
}

const copyText = async (text) => {
  const t = String(text ?? '')
  if (!t) return
  try {
    await navigator.clipboard.writeText(t)
    showSuccess('已复制到剪贴板')
  } catch {
    showErr('无法写入剪贴板，请手动复制')
  }
}

const loadStatus = async () => {
  clearError()
  try {
    status.value = await GetRepoStatus()
  } catch (e) {
    status.value = null
    showErr(e)
  }
}

const doReloadConfig = async () => {
  clearError()
  lockVault()
  try {
    await ReloadConfig()
    await refreshAppConfig()
    await loadStatus()
    showSuccess('配置已重新加载')
  } catch (e) {
    await refreshAppConfig()
    showErr(e)
  }
}

const syncing = ref(false)

const doPull = async () => {
  clearError()
  syncing.value = true
  try {
    await Pull()
    await loadStatus()
    showSuccess('Pull 完成')
  } catch (e) {
    showErr(e)
  } finally {
    syncing.value = false
  }
}

const doPush = async () => {
  clearError()
  syncing.value = true
  try {
    await Push()
    await loadStatus()
    showSuccess('Push 完成')
  } catch (e) {
    showErr(e)
  } finally {
    syncing.value = false
  }
}

const doSync = async () => {
  clearError()
  const password = await askPasswordAsync(
    `输入主密码（用于解密 / 加密 ${vaultLabel.value}）`,
  )
  if (password == null) return
  syncing.value = true
  try {
    await Sync(password)
    await loadStatus()
    if (vaultUnlocked.value) {
      sessionPassword.value = password
      await refreshVaultEntries()
    }
    showSuccess('同步完成')
  } catch (e) {
    showErr(e)
  } finally {
    syncing.value = false
  }
}

const doBindRepo = async () => {
  clearError()
  if (!appConfig.value.remote_url?.trim()) {
    showErr('请先在 pwdmgr.config.json 中填写 remote_url')
    return
  }
  const password = await askPasswordAsync('输入主密码（新建或合并加密库时使用）')
  if (password == null) return
  syncing.value = true
  try {
    await BindRepo(password)
    await loadStatus()
    if (vaultUnlocked.value) {
      sessionPassword.value = password
      await refreshVaultEntries()
    }
    showSuccess('仓库绑定完成')
  } catch (e) {
    showErr(e)
  } finally {
    syncing.value = false
  }
}

const doInitLocalVault = async () => {
  clearError()
  const password = await askPasswordAsync(
    `设置主密码（用于加密 ${vaultLabel.value}）`,
  )
  if (password == null) return
  try {
    await InitLocalVault(password)
    await loadStatus()
    sessionPassword.value = password
    vaultUnlocked.value = true
    entries.value = await ListVaultEntries(password)
    resetVaultForm()
    showSuccess('本地加密库已创建')
  } catch (e) {
    showErr(e)
  }
}

const statusText = () => {
  if (status.value == null) return '（配置有效时点击「刷新状态」）'
  return JSON.stringify(status.value, null, 2)
}

const statusDotClass = computed(() => {
  if (!status.value) return 'dot--idle'
  if (status.value.has_uncommitted || status.value.ahead > 0) return 'dot--warn'
  return 'dot--ok'
})

onMounted(async () => {
  applyWindowChrome()
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  const onSchemeChange = () => applyWindowChrome()
  mq.addEventListener('change', onSchemeChange)
  onUnmounted(() => mq.removeEventListener('change', onSchemeChange))

  await refreshAppConfig()
  if (!appConfig.value.load_error) {
    await loadStatus()
  }
})
</script>

<template>
  <div class="app">
    <!-- Top bar -->
    <header class="topbar">
      <div class="topbar__left">
        <svg class="topbar__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        <span class="topbar__title">PwdMgr</span>
        <span class="topbar__badge">AES-GCM</span>
      </div>
      <div class="topbar__right">
        <span v-if="vaultUnlocked" class="status-chip status-chip--unlocked" @click="lockVault">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>
          已解锁
        </span>
        <span v-else class="status-chip status-chip--locked">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          已锁定
        </span>
      </div>
    </header>

    <!-- Tab navigation -->
    <nav class="tabs">
      <button
        :class="['tabs__btn', activeTab === 'vault' && 'tabs__btn--active']"
        @click="activeTab = 'vault'"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
        保险库
      </button>
      <button
        :class="['tabs__btn', activeTab === 'sync' && 'tabs__btn--active']"
        @click="activeTab = 'sync'"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        同步
        <span v-if="syncing" class="tabs__spinner" />
      </button>
      <button
        :class="['tabs__btn', activeTab === 'settings' && 'tabs__btn--active']"
        @click="activeTab = 'settings'"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        设置
      </button>
    </nav>

    <!-- Main content -->
    <main class="content">
      <!-- ===== Vault Tab ===== -->
      <div v-show="activeTab === 'vault'" class="page">
        <!-- Unlock prompt -->
        <div v-if="!vaultUnlocked" class="unlock-card">
          <div class="unlock-card__icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/><circle cx="12" cy="16" r="1"/></svg>
          </div>
          <h2 class="unlock-card__title">保险库已锁定</h2>
          <p class="unlock-card__desc">输入主密码解锁以查看和管理您的密码条目。密码仅在当前会话中保存于内存。</p>
          <button type="button" class="btn btn--primary btn--lg" @click="doUnlockVault">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>
            解锁保险库
          </button>
        </div>

        <!-- Vault content -->
        <template v-if="vaultUnlocked">
          <!-- Toolbar -->
          <div class="toolbar">
            <div class="search">
              <svg class="search__icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input
                v-model="searchQuery"
                type="text"
                class="search__input"
                placeholder="搜索条目名称、用户名或标签…"
                autocomplete="off"
              />
              <button v-if="searchQuery" type="button" class="search__clear" @click="searchQuery = ''">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <div class="toolbar__actions">
              <button type="button" class="btn btn--ghost btn--icon" @click="refreshVaultEntries" title="刷新条目">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
              </button>
              <button
                type="button"
                class="btn btn--primary"
                @click="formOpen = true; resetVaultForm()"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                添加条目
              </button>
            </div>
          </div>

          <!-- Tag filter -->
          <Transition name="slide">
            <div v-if="allTags.length > 0" class="tag-filter">
              <div class="tag-filter__head">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>
                <span class="tag-filter__label">按标签筛选</span>
                <button
                  v-if="selectedTags.length > 0"
                  type="button"
                  class="tag-filter__clear"
                  @click="clearTagFilter"
                >
                  清除 ({{ selectedTags.length }})
                </button>
              </div>
              <div class="tag-filter__list">
                <button
                  v-for="t in allTags"
                  :key="t"
                  :class="['tag-chip', selectedTags.includes(t) && 'tag-chip--active']"
                  type="button"
                  @click="toggleTag(t)"
                >
                  {{ t }}
                  <svg v-if="selectedTags.includes(t)" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                </button>
              </div>
            </div>
          </Transition>

          <!-- Entry form (slide panel) -->
          <Transition name="slide">
            <div v-if="formOpen" class="form-panel">
              <div class="form-panel__head">
                <h3 class="form-panel__title">{{ formMode === 'add' ? '新建条目' : '编辑条目' }}</h3>
                <button type="button" class="btn--close" @click="formOpen = false; resetVaultForm()">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                </button>
              </div>
              <div class="form-panel__body">
                <div class="form-grid">
                  <label class="field">
                    <span class="field__label">名称</span>
                    <input v-model="form.name" class="field__input" type="text" placeholder="例如：GitHub" autocomplete="off" />
                  </label>
                  <label class="field">
                    <span class="field__label">用户名</span>
                    <input v-model="form.username" class="field__input" type="text" placeholder="用户名或邮箱" autocomplete="off" />
                  </label>
                  <label class="field field--full">
                    <span class="field__label">密码</span>
                    <input v-model="form.password" class="field__input field__input--mono" type="text" placeholder="密码" autocomplete="off" />
                  </label>
                  <label class="field field--full">
                    <span class="field__label">标签</span>
                    <input v-model="form.tagsStr" class="field__input" type="text" placeholder="多个用逗号分隔，例如：工作, 社交" autocomplete="off" />
                  </label>
                  <label class="field field--full">
                    <span class="field__label">备注</span>
                    <textarea v-model="form.note" class="field__input field__textarea" rows="2" placeholder="可选备注信息" />
                  </label>
                </div>
                <div class="form-panel__actions">
                  <button type="button" class="btn btn--ghost" @click="formOpen = false; resetVaultForm()">取消</button>
                  <button type="button" class="btn btn--primary" @click="submitVaultForm">
                    {{ formMode === 'add' ? '添加' : '保存修改' }}
                  </button>
                </div>
              </div>
            </div>
          </Transition>

          <!-- Entries list -->
          <div class="entries">
            <TransitionGroup name="entry">
              <div v-for="row in filteredEntries" :key="row.id" class="entry-card">
                <div class="entry-card__main">
                  <div class="entry-card__info">
                    <div class="entry-card__avatar">{{ (row.name || '?')[0].toUpperCase() }}</div>
                    <div class="entry-card__meta">
                      <span class="entry-card__name">{{ row.name }}</span>
                      <span class="entry-card__user">{{ row.username || '—' }}</span>
                    </div>
                  </div>
                  <div class="entry-card__tags" v-if="row.tags && row.tags.length">
                    <button
                      v-for="t in row.tags"
                      :key="t"
                      :class="['tag', selectedTags.includes(t) && 'tag--active']"
                      type="button"
                      @click.stop="toggleTag(t)"
                      :title="`按标签「${t}」筛选`"
                    >{{ t }}</button>
                  </div>
                </div>
                <div class="entry-card__pwd">
                  <code class="entry-card__pwd-text">{{ pwdVisible[row.id] ? row.password : '••••••••••' }}</code>
                  <div class="entry-card__pwd-actions">
                    <button type="button" class="icon-btn" @click="togglePwdRow(row.id)" :title="pwdVisible[row.id] ? '隐藏密码' : '显示密码'">
                      <svg v-if="!pwdVisible[row.id]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                    </button>
                    <button type="button" class="icon-btn" @click="copyText(row.password)" title="复制密码">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    </button>
                    <button type="button" class="icon-btn" @click="copyText(row.username)" title="复制用户名">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                    </button>
                  </div>
                </div>
                <div class="entry-card__footer">
                  <button type="button" class="btn btn--ghost btn--sm" @click="startEditEntry(row)">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                    编辑
                  </button>
                  <button type="button" class="btn btn--danger btn--sm" @click="deleteEntry(row)">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                    删除
                  </button>
                </div>
              </div>
            </TransitionGroup>

            <div v-if="filteredEntries.length === 0 && entries.length > 0" class="empty-state">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <p v-if="searchQuery && selectedTags.length">没有同时匹配「{{ searchQuery }}」和所选标签的条目</p>
              <p v-else-if="searchQuery">没有匹配「{{ searchQuery }}」的条目</p>
              <p v-else>没有匹配所选标签的条目</p>
              <button
                v-if="selectedTags.length || searchQuery"
                type="button"
                class="btn btn--ghost btn--sm"
                @click="clearTagFilter(); searchQuery = ''"
              >清除所有筛选</button>
            </div>

            <div v-if="entries.length === 0" class="empty-state">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
              <p>保险库为空</p>
              <span class="empty-state__hint">点击「添加条目」创建第一条密码记录</span>
            </div>
          </div>
        </template>
      </div>

      <!-- ===== Sync Tab ===== -->
      <div v-show="activeTab === 'sync'" class="page">
        <div class="section-header">
          <h2 class="section-header__title">Git 同步</h2>
          <p class="section-header__desc">通过 Git 仓库同步您的加密保险库。支持 Pull、Push 和双向合并同步。</p>
        </div>

        <div class="sync-grid">
          <div class="sync-card">
            <div class="sync-card__icon sync-card__icon--blue">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
            </div>
            <h3 class="sync-card__title">完整同步</h3>
            <p class="sync-card__desc">Pull → 合并加密库 → Commit → Push，一键完成双向同步。</p>
            <button type="button" class="btn btn--primary" :disabled="syncing" @click="doSync">
              {{ syncing ? '同步中…' : 'Sync' }}
            </button>
          </div>

          <div class="sync-card">
            <div class="sync-card__icon sync-card__icon--green">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 20 1 14 7 14"/><path d="M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
            </div>
            <h3 class="sync-card__title">Pull</h3>
            <p class="sync-card__desc">从远程仓库拉取最新更改到本地。</p>
            <button type="button" class="btn btn--ghost" :disabled="syncing" @click="doPull">Pull</button>
          </div>

          <div class="sync-card">
            <div class="sync-card__icon sync-card__icon--orange">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M23 10l-4.64-4.36A9 9 0 0 0 3.51 9"/></svg>
            </div>
            <h3 class="sync-card__title">Push</h3>
            <p class="sync-card__desc">将本地提交推送到远程仓库。</p>
            <button type="button" class="btn btn--ghost" :disabled="syncing" @click="doPush">Push</button>
          </div>
        </div>

        <div class="section-header" style="margin-top: 28px;">
          <h2 class="section-header__title">仓库初始化</h2>
        </div>

        <div class="init-row">
          <button type="button" class="btn btn--accent" :disabled="syncing" @click="doInitLocalVault">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
            创建本地库
          </button>
          <button type="button" class="btn btn--accent" :disabled="syncing" @click="doBindRepo">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
            绑定远程并同步
          </button>
        </div>

        <!-- Repo status -->
        <div class="status-block">
          <div class="status-block__head">
            <div class="status-block__left">
              <span :class="['dot', statusDotClass]" />
              <span class="status-block__title">仓库状态</span>
            </div>
            <button type="button" class="btn btn--ghost btn--sm" @click="loadStatus">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
              刷新
            </button>
          </div>
          <pre class="status-block__pre">{{ statusText() }}</pre>
        </div>
      </div>

      <!-- ===== Settings Tab ===== -->
      <div v-show="activeTab === 'settings'" class="page">
        <div class="section-header">
          <h2 class="section-header__title">应用配置</h2>
          <p class="section-header__desc">
            仓库路径与远程地址由工作目录下的 <code>pwdmgr.config.json</code> 提供。
            环境变量 <code>PWDMGR_CONFIG</code> 可指向任意路径的配置文件。
          </p>
        </div>

        <div v-if="appConfig.load_error" class="alert" role="alert">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
          {{ appConfig.load_error }}
        </div>

        <div class="config-list">
          <div class="config-item">
            <div class="config-item__label">配置文件</div>
            <div class="config-item__value">{{ appConfig.config_path || '—' }}</div>
          </div>
          <div class="config-item">
            <div class="config-item__label">仓库根目录</div>
            <div class="config-item__value">{{ appConfig.repo_root || '—' }}</div>
          </div>
          <div class="config-item">
            <div class="config-item__label">远程 URL</div>
            <div class="config-item__value">{{ appConfig.remote_url || '（未填写，仅本地操作）' }}</div>
          </div>
          <div class="config-item">
            <div class="config-item__label">加密库文件名</div>
            <div class="config-item__value"><code>{{ appConfig.vault_file_name }}</code></div>
          </div>
        </div>

        <div style="margin-top: 20px;">
          <button type="button" class="btn btn--ghost" @click="doReloadConfig">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
            重新加载配置
          </button>
        </div>
      </div>
    </main>

    <!-- Toast notification -->
    <Transition name="toast">
      <div v-if="toastVisible" :class="['toast', `toast--${toastType}`]" @click="toastVisible = false">
        <svg v-if="toastType === 'success'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <span class="toast__msg">{{ toastMsg }}</span>
      </div>
    </Transition>

    <!-- Password dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="dlgPwdOpen" class="modal-backdrop" role="presentation" @click.self="dlgPwdCancel">
          <div class="modal" role="dialog" aria-modal="true" @keydown.escape.prevent="dlgPwdCancel">
            <div class="modal__icon">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/><circle cx="12" cy="16" r="1"/></svg>
            </div>
            <h3 class="modal__title">{{ dlgPwdTitle }}</h3>
            <input
              ref="dlgPwdInputRef"
              v-model="dlgPwdValue"
              type="password"
              class="field__input"
              placeholder="输入主密码"
              autocomplete="current-password"
              @keydown.enter.prevent="dlgPwdConfirm"
              @keydown.escape.prevent="dlgPwdCancel"
            />
            <p class="modal__hint">留空并确定视为取消</p>
            <div class="modal__actions">
              <button type="button" class="btn btn--ghost" @click="dlgPwdCancel">取消</button>
              <button type="button" class="btn btn--primary" @click="dlgPwdConfirm">确定</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Confirm dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="dlgConfirmOpen" class="modal-backdrop" role="presentation" @click.self="dlgConfirmNo">
          <div class="modal modal--narrow" role="dialog" aria-modal="true" @keydown.escape.prevent="dlgConfirmNo">
            <div class="modal__icon modal__icon--warn">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
            </div>
            <p class="modal__text">{{ dlgConfirmMsg }}</p>
            <div class="modal__actions">
              <button type="button" class="btn btn--ghost" @click="dlgConfirmNo">取消</button>
              <button type="button" class="btn btn--danger" @click="dlgConfirmYes">确定</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
/* ===== Layout ===== */
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* ===== Top bar ===== */
.topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: color-mix(in srgb, Canvas 85%, transparent);
  backdrop-filter: blur(16px) saturate(180%);
  -webkit-backdrop-filter: blur(16px) saturate(180%);
  border-bottom: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  --wails-draggable: drag;
}

.topbar__left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.topbar__icon {
  width: 22px;
  height: 22px;
  color: Highlight;
}

.topbar__title {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.topbar__badge {
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  background: color-mix(in srgb, CanvasText 6%, transparent);
}

.topbar__right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  cursor: default;
  transition: all 0.2s ease;
}

.status-chip--unlocked {
  color: #16a34a;
  background: color-mix(in srgb, #16a34a 12%, transparent);
  cursor: pointer;
}

.status-chip--unlocked:hover {
  background: color-mix(in srgb, #16a34a 18%, transparent);
}

.status-chip--locked {
  color: color-mix(in srgb, CanvasText 55%, transparent);
  background: color-mix(in srgb, CanvasText 6%, transparent);
}

/* ===== Tabs ===== */
.tabs {
  display: flex;
  gap: 2px;
  padding: 0 20px;
  background: color-mix(in srgb, Canvas 92%, transparent);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
}

.tabs__btn {
  appearance: none;
  border: none;
  background: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 12px 16px;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  border-bottom: 2px solid transparent;
  transition: all 0.2s ease;
  position: relative;
}

.tabs__btn:hover {
  color: color-mix(in srgb, CanvasText 80%, transparent);
  background: color-mix(in srgb, CanvasText 3%, transparent);
}

.tabs__btn--active {
  color: Highlight;
  border-bottom-color: Highlight;
  font-weight: 600;
}

.tabs__spinner {
  width: 12px;
  height: 12px;
  border: 2px solid color-mix(in srgb, Highlight 30%, transparent);
  border-top-color: Highlight;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ===== Content ===== */
.content {
  flex: 1;
  padding: 24px 20px 40px;
  max-width: 720px;
  width: 100%;
  margin: 0 auto;
}

.page {
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== Unlock card ===== */
.unlock-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 60px 24px;
}

.unlock-card__icon {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 24px;
  background: color-mix(in srgb, Highlight 10%, transparent);
  color: Highlight;
  margin-bottom: 24px;
}

.unlock-card__title {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.unlock-card__desc {
  margin: 0 0 28px;
  max-width: 36ch;
  font-size: 14px;
  line-height: 1.6;
  color: color-mix(in srgb, CanvasText 60%, transparent);
}

/* ===== Toolbar ===== */
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.search {
  flex: 1;
  position: relative;
}

.search__icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: color-mix(in srgb, CanvasText 40%, transparent);
  pointer-events: none;
}

.search__input {
  width: 100%;
  padding: 10px 36px 10px 38px;
  font: inherit;
  font-size: 13px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, CanvasText 12%, transparent);
  background: color-mix(in srgb, Canvas 60%, CanvasText 2%);
  color: CanvasText;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.search__input::placeholder {
  color: color-mix(in srgb, CanvasText 38%, transparent);
}

.search__input:focus {
  border-color: color-mix(in srgb, Highlight 50%, CanvasText 20%);
  box-shadow: 0 0 0 3px color-mix(in srgb, Highlight 15%, transparent);
}

.search__clear {
  appearance: none;
  border: none;
  background: none;
  cursor: pointer;
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  padding: 4px;
  border-radius: 6px;
  color: color-mix(in srgb, CanvasText 45%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
}

.search__clear:hover {
  background: color-mix(in srgb, CanvasText 8%, transparent);
  color: CanvasText;
}

.toolbar__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* ===== Tag filter ===== */
.tag-filter {
  margin-bottom: 16px;
  padding: 12px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
  border: 1px solid color-mix(in srgb, CanvasText 7%, transparent);
}

@media (prefers-color-scheme: dark) {
  .tag-filter {
    background: color-mix(in srgb, Canvas 82%, CanvasText 18%);
  }
}

.tag-filter__head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
}

.tag-filter__label {
  font-size: 12px;
  font-weight: 600;
  flex: 1;
}

.tag-filter__clear {
  appearance: none;
  border: none;
  background: none;
  cursor: pointer;
  font: inherit;
  font-size: 11px;
  font-weight: 600;
  color: Highlight;
  padding: 2px 6px;
  border-radius: 4px;
  transition: background 0.15s;
}

.tag-filter__clear:hover {
  background: color-mix(in srgb, Highlight 10%, transparent);
}

.tag-filter__list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-chip {
  appearance: none;
  border: none;
  cursor: pointer;
  font: inherit;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  color: color-mix(in srgb, CanvasText 70%, transparent);
  background: color-mix(in srgb, CanvasText 6%, transparent);
  transition: all 0.15s ease;
}

.tag-chip:hover {
  background: color-mix(in srgb, CanvasText 10%, transparent);
  color: CanvasText;
}

.tag-chip--active {
  color: #fff;
  background: Highlight;
}

.tag-chip--active:hover {
  filter: brightness(1.08);
  color: #fff;
}

/* ===== Form panel ===== */
.form-panel {
  margin-bottom: 20px;
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, Highlight 20%, color-mix(in srgb, CanvasText 10%, transparent));
  background: color-mix(in srgb, Highlight 3%, Canvas);
  overflow: hidden;
}

.form-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
}

.form-panel__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.btn--close {
  appearance: none;
  border: none;
  background: none;
  cursor: pointer;
  padding: 4px;
  border-radius: 8px;
  color: color-mix(in srgb, CanvasText 50%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.btn--close:hover {
  background: color-mix(in srgb, CanvasText 8%, transparent);
  color: CanvasText;
}

.form-panel__body {
  padding: 18px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.form-panel__actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
}

/* ===== Fields ===== */
.field {
  display: block;
}

.field--full {
  grid-column: 1 / -1;
}

.field__label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 6px;
  color: color-mix(in srgb, CanvasText 65%, transparent);
}

.field__input {
  width: 100%;
  padding: 9px 12px;
  font: inherit;
  font-size: 13px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, CanvasText 14%, transparent);
  background: Canvas;
  color: CanvasText;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field__input--mono {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.field__input::placeholder {
  color: color-mix(in srgb, CanvasText 35%, transparent);
}

.field__input:hover {
  border-color: color-mix(in srgb, CanvasText 22%, transparent);
}

.field__input:focus {
  border-color: color-mix(in srgb, Highlight 50%, CanvasText 20%);
  box-shadow: 0 0 0 3px color-mix(in srgb, Highlight 15%, transparent);
}

.field__textarea {
  resize: vertical;
  min-height: 48px;
}

/* ===== Entry cards ===== */
.entries {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.entry-card {
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
  padding: 16px;
  transition: all 0.2s ease;
}

.entry-card:hover {
  border-color: color-mix(in srgb, CanvasText 14%, transparent);
  box-shadow: 0 4px 16px color-mix(in srgb, CanvasText 6%, transparent);
}

@media (prefers-color-scheme: dark) {
  .entry-card {
    background: color-mix(in srgb, Canvas 80%, CanvasText 20%);
  }
}

.entry-card__main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.entry-card__info {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.entry-card__avatar {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: Highlight;
  background: color-mix(in srgb, Highlight 12%, transparent);
  flex-shrink: 0;
}

.entry-card__meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.entry-card__name {
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-card__user {
  font-size: 12px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.entry-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  flex-shrink: 0;
}

.tag {
  appearance: none;
  border: none;
  cursor: pointer;
  font: inherit;
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  background: color-mix(in srgb, Highlight 10%, transparent);
  color: color-mix(in srgb, CanvasText 75%, Highlight 25%);
  transition: all 0.15s ease;
}

.tag:hover {
  background: color-mix(in srgb, Highlight 18%, transparent);
}

.tag--active {
  background: color-mix(in srgb, Highlight 85%, CanvasText 15%);
  color: #fff;
}

.tag--active:hover {
  filter: brightness(1.08);
}

.entry-card__pwd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  background: color-mix(in srgb, CanvasText 4%, transparent);
  margin-bottom: 12px;
}

.entry-card__pwd-text {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  letter-spacing: 0.02em;
  color: color-mix(in srgb, CanvasText 80%, transparent);
  user-select: all;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-card__pwd-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.icon-btn {
  appearance: none;
  border: none;
  background: none;
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  color: color-mix(in srgb, CanvasText 50%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.icon-btn:hover {
  background: color-mix(in srgb, CanvasText 8%, transparent);
  color: CanvasText;
}

.entry-card__footer {
  display: flex;
  gap: 8px;
}

/* ===== Empty state ===== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
  color: color-mix(in srgb, CanvasText 40%, transparent);
}

.empty-state p {
  margin: 12px 0 4px;
  font-size: 15px;
  font-weight: 500;
  color: color-mix(in srgb, CanvasText 60%, transparent);
}

.empty-state__hint {
  font-size: 13px;
}

/* ===== Section header ===== */
.section-header {
  margin-bottom: 20px;
}

.section-header__title {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.section-header__desc {
  margin: 0;
  font-size: 13px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  line-height: 1.6;
}

.section-header__desc code {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: color-mix(in srgb, CanvasText 7%, transparent);
}

/* ===== Sync grid ===== */
.sync-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

@media (max-width: 600px) {
  .sync-grid { grid-template-columns: 1fr; }
}

.sync-card {
  padding: 20px;
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@media (prefers-color-scheme: dark) {
  .sync-card {
    background: color-mix(in srgb, Canvas 80%, CanvasText 20%);
  }
}

.sync-card__icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}

.sync-card__icon--blue {
  background: color-mix(in srgb, #3b82f6 12%, transparent);
  color: #3b82f6;
}

.sync-card__icon--green {
  background: color-mix(in srgb, #16a34a 12%, transparent);
  color: #16a34a;
}

.sync-card__icon--orange {
  background: color-mix(in srgb, #ea580c 12%, transparent);
  color: #ea580c;
}

.sync-card__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.sync-card__desc {
  margin: 0;
  font-size: 12px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  line-height: 1.5;
  flex: 1;
}

.sync-card .btn {
  width: 100%;
  justify-content: center;
  margin-top: 4px;
}

.init-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 28px;
}

/* ===== Status block ===== */
.status-block {
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  overflow: hidden;
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
}

@media (prefers-color-scheme: dark) {
  .status-block {
    background: color-mix(in srgb, Canvas 80%, CanvasText 20%);
  }
}

.status-block__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
}

.status-block__left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot--ok { background: #16a34a; box-shadow: 0 0 6px color-mix(in srgb, #16a34a 40%, transparent); }
.dot--warn { background: #ea580c; box-shadow: 0 0 6px color-mix(in srgb, #ea580c 40%, transparent); }
.dot--idle { background: color-mix(in srgb, CanvasText 30%, transparent); }

.status-block__title {
  font-size: 13px;
  font-weight: 600;
}

.status-block__pre {
  margin: 0;
  padding: 14px 16px;
  max-height: 300px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  tab-size: 2;
  color: color-mix(in srgb, CanvasText 80%, transparent);
}

/* ===== Config list ===== */
.config-list {
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  overflow: hidden;
}

.config-item {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 5%, transparent);
  transition: background 0.15s;
}

.config-item:last-child {
  border-bottom: none;
}

.config-item:hover {
  background: color-mix(in srgb, CanvasText 2%, transparent);
}

.config-item__label {
  font-size: 12px;
  font-weight: 600;
  color: color-mix(in srgb, CanvasText 60%, transparent);
  padding-top: 1px;
}

.config-item__value {
  font-size: 13px;
  line-height: 1.5;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.config-item__value code {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
  background: color-mix(in srgb, CanvasText 7%, transparent);
}

/* ===== Alert ===== */
.alert {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 20px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.5;
  color: #dc2626;
  background: color-mix(in srgb, #dc2626 8%, Canvas);
  border: 1px solid color-mix(in srgb, #dc2626 20%, transparent);
}

.alert svg {
  flex-shrink: 0;
  margin-top: 1px;
}

/* ===== Buttons ===== */
.btn {
  appearance: none;
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  padding: 8px 16px;
  border-radius: 10px;
  border: 1px solid transparent;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.btn:focus-visible {
  outline: 2px solid color-mix(in srgb, Highlight 60%, transparent);
  outline-offset: 2px;
}

.btn:active {
  transform: scale(0.97);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.btn--lg {
  padding: 12px 28px;
  font-size: 15px;
  border-radius: 12px;
}

.btn--sm {
  padding: 5px 10px;
  font-size: 12px;
  border-radius: 8px;
}

.btn--ghost {
  color: CanvasText;
  background: color-mix(in srgb, CanvasText 5%, Canvas);
  border-color: color-mix(in srgb, CanvasText 10%, transparent);
}

.btn--ghost:hover:not(:disabled) {
  background: color-mix(in srgb, CanvasText 10%, Canvas);
  border-color: color-mix(in srgb, CanvasText 16%, transparent);
}

.btn--primary {
  color: #fff;
  background: Highlight;
  border-color: color-mix(in srgb, Highlight 80%, CanvasText 20%);
}

.btn--primary:hover:not(:disabled) {
  filter: brightness(1.08);
}

.btn--accent {
  color: #fff;
  background: color-mix(in srgb, Highlight 85%, CanvasText 15%);
  border-color: color-mix(in srgb, Highlight 65%, CanvasText 35%);
}

.btn--accent:hover:not(:disabled) {
  filter: brightness(1.06);
}

.btn--danger {
  color: #fff;
  background: #dc2626;
  border-color: #b91c1c;
}

.btn--danger:hover:not(:disabled) {
  background: #b91c1c;
}

.btn--icon {
  padding: 8px;
}

/* ===== Toast ===== */
.toast {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 20000;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 8px 32px color-mix(in srgb, CanvasText 15%, transparent);
}

.toast--error {
  color: #fff;
  background: color-mix(in srgb, #dc2626 92%, transparent);
}

.toast--success {
  color: #fff;
  background: color-mix(in srgb, #16a34a 92%, transparent);
}

.toast__msg {
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(16px);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-8px);
}

/* ===== Modal ===== */
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: color-mix(in srgb, CanvasText 40%, transparent);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
}

.modal {
  width: 100%;
  max-width: 400px;
  padding: 28px;
  border-radius: 18px;
  background: Canvas;
  color: CanvasText;
  border: 1px solid color-mix(in srgb, CanvasText 10%, transparent);
  box-shadow: 0 24px 64px color-mix(in srgb, CanvasText 20%, transparent);
}

.modal--narrow {
  max-width: 360px;
}

.modal__icon {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: color-mix(in srgb, Highlight 10%, transparent);
  color: Highlight;
  margin-bottom: 18px;
}

.modal__icon--warn {
  background: color-mix(in srgb, #ea580c 10%, transparent);
  color: #ea580c;
}

.modal__title {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

.modal__text {
  margin: 0 0 20px;
  font-size: 14px;
  line-height: 1.5;
}

.modal__hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: color-mix(in srgb, CanvasText 45%, transparent);
}

.modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 22px;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal,
.modal-leave-active .modal {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal {
  transform: scale(0.95) translateY(8px);
}

.modal-leave-to .modal {
  transform: scale(0.97) translateY(4px);
}

/* ===== Slide transition for form ===== */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  max-height: 0;
  margin-bottom: 0;
  transform: translateY(-8px);
}

.slide-enter-to,
.slide-leave-from {
  max-height: 500px;
}

/* ===== Entry list transitions ===== */
.entry-enter-active,
.entry-leave-active {
  transition: all 0.25s ease;
}

.entry-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.entry-leave-to {
  opacity: 0;
  transform: translateX(-16px);
}

.entry-move {
  transition: transform 0.25s ease;
}

/* ===== Responsive ===== */
@media (max-width: 520px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar__actions {
    justify-content: flex-end;
  }

  .entry-card__main {
    flex-direction: column;
  }

  .config-item {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
