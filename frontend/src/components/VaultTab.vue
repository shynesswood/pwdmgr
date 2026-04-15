<script setup>
import { ref, computed, inject, nextTick } from 'vue'
import {
  ListVaultEntries,
  AddVaultEntry,
  UpdateVaultEntry,
  DeleteVaultEntry,
} from '../../wailsjs/go/app/App'
import EntryCard from './EntryCard.vue'
import EntryFormPanel from './EntryFormPanel.vue'

const props = defineProps({
  vaultLabel: String,
})

const emit = defineEmits(['update:unlocked'])

const askPassword = inject('askPassword')
const askConfirm = inject('askConfirm')
const showErr = inject('showErr')
const showSuccess = inject('showSuccess')

const vaultUnlocked = ref(false)
const sessionPassword = ref('')
const entries = ref([])

const searchQuery = ref('')
const selectedTags = ref([])
const formMode = ref('add')
const formOpen = ref(false)
const form = ref({ id: '', name: '', username: '', password: '', note: '', tagsStr: '' })
const pwdVisible = ref({})

const allTags = computed(() => {
  const set = new Set()
  for (const e of entries.value || []) {
    for (const t of e.tags || []) set.add(t)
  }
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-Hans'))
})

const filteredEntries = computed(() => {
  let list = entries.value || []
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

const parseTags = (s) =>
  String(s || '').split(/[,，]/).map((t) => t.trim()).filter(Boolean)

const resetForm = () => {
  formMode.value = 'add'
  form.value = { id: '', name: '', username: '', password: '', note: '', tagsStr: '' }
}

const toggleTag = (tag) => {
  const idx = selectedTags.value.indexOf(tag)
  if (idx === -1) {
    selectedTags.value = [...selectedTags.value, tag]
  } else {
    selectedTags.value = selectedTags.value.filter((t) => t !== tag)
  }
}

const clearTagFilter = () => { selectedTags.value = [] }
const togglePwdRow = (id) => { pwdVisible.value = { ...pwdVisible.value, [id]: !pwdVisible.value[id] } }

const setUnlocked = (val) => {
  vaultUnlocked.value = val
  emit('update:unlocked', val)
}

const lockVault = () => {
  setUnlocked(false)
  sessionPassword.value = ''
  entries.value = []
  pwdVisible.value = {}
  selectedTags.value = []
  resetForm()
  formOpen.value = false
}

const doUnlock = async () => {
  const p = await askPassword(`输入主密码以打开加密库（${props.vaultLabel}）`)
  if (p == null) return
  try {
    entries.value = (await ListVaultEntries(p)) || []
    sessionPassword.value = p
    setUnlocked(true)
    resetForm()
    showSuccess('保险库已解锁')
  } catch (e) {
    showErr(e)
  }
}

const refreshEntries = async () => {
  if (!vaultUnlocked.value || !sessionPassword.value) return
  try {
    entries.value = (await ListVaultEntries(sessionPassword.value)) || []
  } catch (e) {
    showErr(e)
    lockVault()
  }
}

const doManualRefresh = async () => {
  await refreshEntries()
  if (vaultUnlocked.value) showSuccess('条目已刷新')
}

const refreshWithPassword = async (pwd) => {
  if (!vaultUnlocked.value) return
  try {
    sessionPassword.value = pwd
    entries.value = (await ListVaultEntries(pwd)) || []
  } catch (e) {
    showErr(e)
    lockVault()
  }
}

const startEdit = async (row) => {
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
  await nextTick()
  const el = document.querySelector('.entry-edit-form')
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}

const submitForm = async () => {
  const p = sessionPassword.value
  if (!p) { showErr('请先解锁保险库并输入主密码'); return }
  const { id, name, username, password, note, tagsStr } = form.value
  if (!String(name).trim()) { showErr('请填写条目名称'); return }
  const tags = parseTags(tagsStr)
  try {
    if (formMode.value === 'add') {
      await AddVaultEntry(p, name, username, password, note, tags)
      showSuccess('条目已添加')
    } else {
      await UpdateVaultEntry(p, { id, name, username, password, note, tags, updated_at: 0 })
      showSuccess('条目已更新')
    }
    entries.value = (await ListVaultEntries(p)) || []
    resetForm()
    formOpen.value = false
  } catch (e) {
    showErr(e)
  }
}

const deleteEntry = async (row) => {
  const ok = await askConfirm(`确定删除「${row.name || row.id}」？此操作不可撤销。`)
  if (!ok) return
  try {
    await DeleteVaultEntry(sessionPassword.value, row.id)
    entries.value = (await ListVaultEntries(sessionPassword.value)) || []
    if (formMode.value === 'edit' && form.value.id === row.id) {
      resetForm()
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

defineExpose({ lockVault, refreshWithPassword, refreshEntries })
</script>

<template>
  <div class="page">
    <!-- Unlock prompt -->
    <div v-if="!vaultUnlocked" class="unlock-card">
      <div class="unlock-card__icon">
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/><circle cx="12" cy="16" r="1"/></svg>
      </div>
      <h2 class="unlock-card__title">保险库已锁定</h2>
      <p class="unlock-card__desc">输入主密码解锁以查看和管理您的密码条目。密码仅在当前会话中保存于内存。</p>
      <button type="button" class="btn btn--primary btn--lg" @click="doUnlock">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>
        解锁保险库
      </button>
    </div>

    <!-- Vault content -->
    <template v-if="vaultUnlocked">
      <div class="toolbar">
        <div class="search">
          <svg class="search__icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input v-model="searchQuery" type="text" class="search__input" placeholder="搜索条目名称、用户名或标签…" autocomplete="off" />
          <button v-if="searchQuery" type="button" class="search__clear" @click="searchQuery = ''">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
        <div class="toolbar__actions">
          <button type="button" class="btn btn--ghost btn--icon" @click="doManualRefresh" title="刷新条目">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          </button>
          <button type="button" class="btn btn--primary" @click="formOpen = true; resetForm()">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            添加条目
          </button>
        </div>
      </div>

      <div class="entries-status">
        共 <strong>{{ entries.length }}</strong> 条记录
        <span v-if="filteredEntries.length !== entries.length" class="entries-status__filter">
          · 筛选显示 <strong>{{ filteredEntries.length }}</strong> 条
        </span>
      </div>

      <!-- Tag filter -->
      <Transition name="slide">
        <div v-if="allTags.length > 0" class="tag-filter">
          <div class="tag-filter__head">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>
            <span class="tag-filter__label">按标签筛选</span>
            <button v-if="selectedTags.length > 0" type="button" class="tag-filter__clear" @click="clearTagFilter">
              清除 ({{ selectedTags.length }})
            </button>
          </div>
          <div class="tag-filter__list">
            <button v-for="t in allTags" :key="t" :class="['tag-chip', selectedTags.includes(t) && 'tag-chip--active']" type="button" @click="toggleTag(t)">
              {{ t }}
              <svg v-if="selectedTags.includes(t)" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
            </button>
          </div>
        </div>
      </Transition>

      <!-- Add entry form (top position) -->
      <Transition name="slide">
        <EntryFormPanel
          v-if="formOpen && formMode === 'add'"
          :mode="formMode"
          :form="form"
          class="form-panel-wrapper"
          @submit="submitForm"
          @cancel="formOpen = false; resetForm()"
        />
      </Transition>

      <!-- Entries list -->
      <div class="entries">
        <template v-for="row in filteredEntries" :key="row.id">
          <EntryCard
            :entry="row"
            :pwd-visible="!!pwdVisible[row.id]"
            :selected-tags="selectedTags"
            @toggle-pwd="togglePwdRow(row.id)"
            @copy="copyText($event)"
            @edit="startEdit(row)"
            @delete="deleteEntry(row)"
            @toggle-tag="toggleTag($event)"
          />
          <Transition name="slide">
            <EntryFormPanel
              v-if="formOpen && formMode === 'edit' && form.id === row.id"
              :mode="formMode"
              :form="form"
              class="entry-edit-form"
              @submit="submitForm"
              @cancel="formOpen = false; resetForm()"
            />
          </Transition>
        </template>

        <div v-if="filteredEntries.length === 0 && entries.length > 0" class="empty-state">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <p v-if="searchQuery && selectedTags.length">没有同时匹配「{{ searchQuery }}」和所选标签的条目</p>
          <p v-else-if="searchQuery">没有匹配「{{ searchQuery }}」的条目</p>
          <p v-else>没有匹配所选标签的条目</p>
          <button v-if="selectedTags.length || searchQuery" type="button" class="btn btn--ghost btn--sm" @click="clearTagFilter(); searchQuery = ''">清除所有筛选</button>
        </div>

        <div v-if="entries.length === 0" class="empty-state">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
          <p>保险库为空</p>
          <span class="empty-state__hint">点击「添加条目」创建第一条密码记录</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.unlock-card { display: flex; flex-direction: column; align-items: center; text-align: center; padding: 40px 24px; }
.unlock-card__icon { width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; border-radius: 18px; background: color-mix(in srgb, Highlight 10%, transparent); color: Highlight; margin-bottom: 18px; }
.unlock-card__title { margin: 0 0 6px; font-size: 18px; font-weight: 700; letter-spacing: -0.02em; }
.unlock-card__desc { margin: 0 0 22px; max-width: 42ch; font-size: 13px; line-height: 1.6; color: color-mix(in srgb, CanvasText 55%, transparent); }

.toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.toolbar__actions { display: flex; gap: 6px; flex-shrink: 0; }

.search { flex: 1; position: relative; }
.search__icon { position: absolute; left: 10px; top: 50%; transform: translateY(-50%); color: color-mix(in srgb, CanvasText 40%, transparent); pointer-events: none; }
.search__input { width: 100%; padding: 8px 32px 8px 34px; font: inherit; font-size: 13px; border-radius: 8px; border: 1px solid color-mix(in srgb, CanvasText 10%, transparent); background: color-mix(in srgb, Canvas 60%, CanvasText 2%); color: CanvasText; outline: none; transition: border-color 0.2s, box-shadow 0.2s; }
.search__input::placeholder { color: color-mix(in srgb, CanvasText 38%, transparent); }
.search__input:focus { border-color: color-mix(in srgb, Highlight 50%, CanvasText 20%); box-shadow: 0 0 0 3px color-mix(in srgb, Highlight 15%, transparent); }
.search__clear { appearance: none; border: none; background: none; cursor: pointer; position: absolute; right: 6px; top: 50%; transform: translateY(-50%); padding: 3px; border-radius: 5px; color: color-mix(in srgb, CanvasText 45%, transparent); display: flex; align-items: center; justify-content: center; }
.search__clear:hover { background: color-mix(in srgb, CanvasText 8%, transparent); color: CanvasText; }

.tag-filter { margin-bottom: 12px; padding: 10px 12px; border-radius: 8px; background: color-mix(in srgb, Canvas 94%, CanvasText 6%); border: 1px solid color-mix(in srgb, CanvasText 5%, transparent); }
@media (prefers-color-scheme: dark) { .tag-filter { background: color-mix(in srgb, Canvas 82%, CanvasText 18%); } }
.tag-filter__head { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; color: color-mix(in srgb, CanvasText 55%, transparent); }
.tag-filter__label { font-size: 11px; font-weight: 600; flex: 1; }
.tag-filter__clear { appearance: none; border: none; background: none; cursor: pointer; font: inherit; font-size: 11px; font-weight: 600; color: Highlight; padding: 2px 6px; border-radius: 4px; transition: background 0.15s; }
.tag-filter__clear:hover { background: color-mix(in srgb, Highlight 10%, transparent); }
.tag-filter__list { display: flex; flex-wrap: wrap; gap: 4px; }

.tag-chip { appearance: none; border: none; cursor: pointer; font: inherit; display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; border-radius: 999px; font-size: 11px; font-weight: 500; color: color-mix(in srgb, CanvasText 70%, transparent); background: color-mix(in srgb, CanvasText 6%, transparent); transition: all 0.15s ease; }
.tag-chip:hover { background: color-mix(in srgb, CanvasText 10%, transparent); color: CanvasText; }
.tag-chip--active { color: #fff; background: Highlight; }
.tag-chip--active:hover { filter: brightness(1.08); color: #fff; }

.entries-status { font-size: 12px; color: color-mix(in srgb, CanvasText 45%, transparent); margin-bottom: 10px; padding: 0 2px; }
.entries-status strong { font-weight: 700; color: color-mix(in srgb, CanvasText 65%, transparent); }
.entries-status__filter { color: Highlight; }
.entries-status__filter strong { color: inherit; }

.form-panel-wrapper { margin-bottom: 14px; }
.entry-edit-form { margin-top: 6px; }

.entries { display: flex; flex-direction: column; gap: 6px; }

.empty-state { display: flex; flex-direction: column; align-items: center; padding: 40px 24px; color: color-mix(in srgb, CanvasText 40%, transparent); }
.empty-state p { margin: 10px 0 4px; font-size: 14px; font-weight: 500; color: color-mix(in srgb, CanvasText 55%, transparent); }
.empty-state__hint { font-size: 12px; }

@media (max-width: 520px) {
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar__actions { justify-content: flex-end; }
}
</style>
