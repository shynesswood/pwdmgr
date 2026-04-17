<script setup>
import { ref, computed, inject, nextTick, watch } from 'vue'
import {
  ListVaultEntries,
  AddVaultEntry,
  UpdateVaultEntry,
  DeleteVaultEntry,
  MoveVaultEntries,
  ListVaultSpaces,
  CreateVaultSpace,
  RenameVaultSpace,
  DeleteVaultSpace,
} from '../../wailsjs/go/app/App'
import EntryCard from './EntryCard.vue'
import EntryFormPanel from './EntryFormPanel.vue'

const DEFAULT_SPACE_ID = 'default'

const props = defineProps({
  vaultLabel: String,
})

const emit = defineEmits(['update:unlocked'])

const askPassword = inject('askPassword')
const askConfirm = inject('askConfirm')
const askSpace = inject('askSpace')
const showErr = inject('showErr')
const showSuccess = inject('showSuccess')

const vaultUnlocked = ref(false)
const sessionPassword = ref('')
const entries = ref([])

const spaces = ref([])
const currentSpaceID = ref(DEFAULT_SPACE_ID)
const spaceFormOpen = ref(false)
const spaceFormMode = ref('add') // 'add' | 'rename'
const spaceFormName = ref('')
const spaceFormTargetID = ref('')

const searchQuery = ref('')
const selectedTags = ref([])
const formMode = ref('add')
const formOpen = ref(false)
const form = ref({ id: '', name: '', username: '', password: '', note: '', tagsStr: '' })
const pwdVisible = ref({})

const selectMode = ref(false)
const selectedIDs = ref(new Set())

const currentSpace = computed(() =>
  spaces.value.find((s) => s.id === currentSpaceID.value) || null,
)

const isDefaultSpaceActive = computed(() => currentSpaceID.value === DEFAULT_SPACE_ID)

const allTags = computed(() => {
  const set = new Set()
  for (const e of entries.value || []) {
    for (const t of e.tags || []) set.add(t)
  }
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-Hans'))
})

const selectedCount = computed(() => selectedIDs.value.size)

const allFilteredSelected = computed(() => {
  if (filteredEntries.value.length === 0) return false
  return filteredEntries.value.every((e) => selectedIDs.value.has(e.id))
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

const resetSpaceForm = () => {
  spaceFormOpen.value = false
  spaceFormMode.value = 'add'
  spaceFormName.value = ''
  spaceFormTargetID.value = ''
}

const clearSelection = () => {
  selectedIDs.value = new Set()
}

const exitSelectMode = () => {
  selectMode.value = false
  clearSelection()
}

const enterSelectMode = () => {
  selectMode.value = true
  clearSelection()
  resetForm()
  formOpen.value = false
  resetSpaceForm()
}

const toggleEntrySelect = (id) => {
  const next = new Set(selectedIDs.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIDs.value = next
}

const toggleSelectAllFiltered = () => {
  if (allFilteredSelected.value) {
    const next = new Set(selectedIDs.value)
    for (const e of filteredEntries.value) next.delete(e.id)
    selectedIDs.value = next
  } else {
    const next = new Set(selectedIDs.value)
    for (const e of filteredEntries.value) next.add(e.id)
    selectedIDs.value = next
  }
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
  spaces.value = []
  currentSpaceID.value = DEFAULT_SPACE_ID
  pwdVisible.value = {}
  selectedTags.value = []
  resetForm()
  resetSpaceForm()
  exitSelectMode()
  formOpen.value = false
}

const loadSpaces = async (pwd) => {
  try {
    spaces.value = (await ListVaultSpaces(pwd)) || []
  } catch (e) {
    spaces.value = []
    throw e
  }
}

const loadEntries = async (pwd) => {
  entries.value = (await ListVaultEntries(pwd, currentSpaceID.value)) || []
}

const doUnlock = async () => {
  const p = await askPassword(`输入主密码以打开加密库（${props.vaultLabel}）`)
  if (p == null) return
  try {
    await loadSpaces(p)
    currentSpaceID.value = DEFAULT_SPACE_ID
    await loadEntries(p)
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
    await loadSpaces(sessionPassword.value)
    // 当前空间可能因合并被删除，回退到默认空间
    if (!spaces.value.some((s) => s.id === currentSpaceID.value)) {
      currentSpaceID.value = DEFAULT_SPACE_ID
    }
    await loadEntries(sessionPassword.value)
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
    await loadSpaces(pwd)
    if (!spaces.value.some((s) => s.id === currentSpaceID.value)) {
      currentSpaceID.value = DEFAULT_SPACE_ID
    }
    await loadEntries(pwd)
  } catch (e) {
    showErr(e)
    lockVault()
  }
}

const switchSpace = async (id) => {
  if (id === currentSpaceID.value) return
  currentSpaceID.value = id
  selectedTags.value = []
  resetForm()
  formOpen.value = false
  resetSpaceForm()
  exitSelectMode()
  try {
    await loadEntries(sessionPassword.value)
  } catch (e) {
    showErr(e)
  }
}

const startCreateSpace = () => {
  spaceFormMode.value = 'add'
  spaceFormName.value = ''
  spaceFormTargetID.value = ''
  spaceFormOpen.value = true
}

const startRenameSpace = () => {
  if (!currentSpace.value || isDefaultSpaceActive.value) return
  spaceFormMode.value = 'rename'
  spaceFormName.value = currentSpace.value.name
  spaceFormTargetID.value = currentSpace.value.id
  spaceFormOpen.value = true
}

const submitSpaceForm = async () => {
  const name = (spaceFormName.value || '').trim()
  if (!name) { showErr('空间名称不能为空'); return }
  try {
    if (spaceFormMode.value === 'add') {
      const created = await CreateVaultSpace(sessionPassword.value, name)
      resetSpaceForm()
      await loadSpaces(sessionPassword.value)
      if (created && created.id) {
        await switchSpace(created.id)
      }
      showSuccess('空间已创建')
    } else {
      await RenameVaultSpace(sessionPassword.value, spaceFormTargetID.value, name)
      resetSpaceForm()
      await loadSpaces(sessionPassword.value)
      showSuccess('空间已重命名')
    }
  } catch (e) {
    showErr(e)
  }
}

const deleteCurrentSpace = async () => {
  if (!currentSpace.value || isDefaultSpaceActive.value) return
  const ok = await askConfirm(`确定删除空间「${currentSpace.value.name}」？只有空的空间才能删除。`)
  if (!ok) return
  try {
    await DeleteVaultSpace(sessionPassword.value, currentSpace.value.id)
    const switchTo = DEFAULT_SPACE_ID
    await loadSpaces(sessionPassword.value)
    await switchSpace(switchTo)
    showSuccess('空间已删除')
  } catch (e) {
    showErr(e)
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
      await AddVaultEntry(p, currentSpaceID.value, name, username, password, note, tags)
      showSuccess('条目已添加')
    } else {
      await UpdateVaultEntry(p, {
        id, name, username, password, note, tags,
        updated_at: 0,
        space_id: currentSpaceID.value,
      })
      showSuccess('条目已更新')
    }
    await loadEntries(p)
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
    await loadEntries(sessionPassword.value)
    if (formMode.value === 'edit' && form.value.id === row.id) {
      resetForm()
      formOpen.value = false
    }
    showSuccess('条目已删除')
  } catch (e) {
    showErr(e)
  }
}

const moveEntriesTo = async (ids) => {
  if (!ids || ids.length === 0) { showErr('请先选择要移动的条目'); return }
  if (spaces.value.length <= 1) {
    showErr('没有其它空间可用，请先新建一个空间')
    return
  }
  const targetID = await askSpace({
    title: ids.length === 1 ? '移动到空间' : `移动 ${ids.length} 条到空间`,
    description: `源空间：${currentSpace.value?.name || '默认空间'}`,
    spaces: spaces.value,
    excludeSpaceID: currentSpaceID.value,
  })
  if (!targetID) return
  try {
    const moved = await MoveVaultEntries(sessionPassword.value, targetID, ids)
    await loadEntries(sessionPassword.value)
    exitSelectMode()
    showSuccess(moved > 0 ? `已移动 ${moved} 条` : '没有条目被移动')
  } catch (e) {
    showErr(e)
  }
}

const moveSingleEntry = (row) => moveEntriesTo([row.id])

const moveSelected = () => moveEntriesTo(Array.from(selectedIDs.value))

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

// 切换空间后若选中的条目已不在当前空间视图中，应该保留还是清空？
// 当前策略：切换空间即退出选择模式，避免跨空间误操作。
watch(currentSpaceID, () => {
  if (selectMode.value) exitSelectMode()
})

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
      <!-- Space switcher -->
      <div class="space-switcher">
        <div class="space-switcher__tabs">
          <button
            v-for="s in spaces"
            :key="s.id"
            type="button"
            :class="['space-tab', currentSpaceID === s.id && 'space-tab--active']"
            @click="switchSpace(s.id)"
          >
            <svg v-if="s.id === 'default'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
            </svg>
            <span>{{ s.name }}</span>
          </button>
          <button
            type="button"
            class="space-tab space-tab--add"
            title="新建空间"
            @click="startCreateSpace"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </button>
        </div>
        <div v-if="!isDefaultSpaceActive" class="space-switcher__actions">
          <button type="button" class="space-action" title="重命名当前空间" @click="startRenameSpace">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 1 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
            重命名
          </button>
          <button type="button" class="space-action space-action--danger" title="删除当前空间" @click="deleteCurrentSpace">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/></svg>
            删除空间
          </button>
        </div>
      </div>

      <!-- Space form (inline, add/rename) -->
      <Transition name="slide">
        <div v-if="spaceFormOpen" class="space-form">
          <div class="space-form__head">
            <span class="space-form__title">{{ spaceFormMode === 'add' ? '新建空间' : '重命名空间' }}</span>
            <button type="button" class="btn--close" @click="resetSpaceForm">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div class="space-form__body">
            <input
              v-model="spaceFormName"
              class="space-form__input"
              type="text"
              maxlength="60"
              placeholder="输入空间名称，例如：工作"
              autocomplete="off"
              @keyup.enter="submitSpaceForm"
            />
            <div class="space-form__actions">
              <button type="button" class="btn btn--ghost btn--sm" @click="resetSpaceForm">取消</button>
              <button type="button" class="btn btn--primary btn--sm" @click="submitSpaceForm">
                {{ spaceFormMode === 'add' ? '创建' : '保存' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>

      <div v-if="!selectMode" class="toolbar">
        <div class="search">
          <svg class="search__icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input v-model="searchQuery" type="text" class="search__input" placeholder="搜索条目名称、用户名或标签…" autocomplete="off" />
          <button v-if="searchQuery" type="button" class="search__clear" @click="searchQuery = ''">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
        <div class="toolbar__actions">
          <button v-if="entries.length > 0" type="button" class="btn btn--ghost btn--icon" @click="enterSelectMode" title="批量选择">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><polyline points="14 17 16 19 20 15"/></svg>
          </button>
          <button type="button" class="btn btn--ghost btn--icon" @click="doManualRefresh" title="刷新条目">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          </button>
          <button type="button" class="btn btn--primary" @click="formOpen = true; resetForm()">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            添加条目
          </button>
        </div>
      </div>

      <!-- Select mode toolbar -->
      <div v-else class="select-toolbar">
        <div class="select-toolbar__info">
          <span class="select-toolbar__count">已选 {{ selectedCount }} 条</span>
          <button type="button" class="select-toolbar__link" @click="toggleSelectAllFiltered">
            {{ allFilteredSelected ? '取消全选' : '全选当前' }}
          </button>
        </div>
        <div class="select-toolbar__actions">
          <button
            type="button"
            class="btn btn--primary btn--sm"
            :disabled="selectedCount === 0"
            @click="moveSelected"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            移动到…
          </button>
          <button type="button" class="btn btn--ghost btn--sm" @click="exitSelectMode">完成</button>
        </div>
      </div>

      <div class="entries-status">
        <span class="entries-status__space">{{ currentSpace?.name || '默认空间' }}</span>
        · 共 <strong>{{ entries.length }}</strong> 条记录
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
            :selectable="selectMode"
            :selected="selectedIDs.has(row.id)"
            @toggle-pwd="togglePwdRow(row.id)"
            @copy="copyText($event)"
            @edit="startEdit(row)"
            @delete="deleteEntry(row)"
            @move="moveSingleEntry(row)"
            @toggle-tag="toggleTag($event)"
            @toggle-select="toggleEntrySelect(row.id)"
          />
          <Transition name="slide">
            <EntryFormPanel
              v-if="!selectMode && formOpen && formMode === 'edit' && form.id === row.id"
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
          <p>{{ currentSpace?.name || '默认空间' }} 为空</p>
          <span class="empty-state__hint">点击「添加条目」在该空间创建第一条密码记录</span>
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

/* ===== Space switcher ===== */
.space-switcher { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
.space-switcher__tabs { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
.space-tab { appearance: none; border: none; cursor: pointer; font: inherit; display: inline-flex; align-items: center; gap: 5px; padding: 5px 12px; border-radius: 999px; font-size: 12px; font-weight: 500; color: color-mix(in srgb, CanvasText 65%, transparent); background: color-mix(in srgb, CanvasText 6%, transparent); transition: all 0.15s ease; }
.space-tab:hover { background: color-mix(in srgb, CanvasText 10%, transparent); color: CanvasText; }
.space-tab--active { color: #fff; background: Highlight; font-weight: 600; }
.space-tab--active:hover { filter: brightness(1.05); color: #fff; }
.space-tab--add { padding: 5px 8px; color: color-mix(in srgb, CanvasText 50%, transparent); }
.space-tab--add:hover { color: Highlight; background: color-mix(in srgb, Highlight 12%, transparent); }

.space-switcher__actions { display: flex; gap: 4px; padding-left: 2px; }
.space-action { appearance: none; border: none; cursor: pointer; font: inherit; display: inline-flex; align-items: center; gap: 4px; padding: 3px 8px; border-radius: 6px; font-size: 11px; font-weight: 500; color: color-mix(in srgb, CanvasText 55%, transparent); background: transparent; transition: all 0.15s ease; }
.space-action:hover { background: color-mix(in srgb, CanvasText 6%, transparent); color: CanvasText; }
.space-action--danger:hover { color: #dc2626; background: color-mix(in srgb, #dc2626 10%, transparent); }

/* Space inline form */
.space-form { margin-bottom: 12px; border-radius: 10px; border: 1px solid color-mix(in srgb, Highlight 20%, color-mix(in srgb, CanvasText 8%, transparent)); background: color-mix(in srgb, Highlight 3%, Canvas); overflow: hidden; }
.space-form__head { display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border-bottom: 1px solid color-mix(in srgb, CanvasText 5%, transparent); }
.space-form__title { font-size: 12px; font-weight: 600; }
.space-form__body { padding: 12px 14px; display: flex; gap: 8px; align-items: center; }
.space-form__input { flex: 1; padding: 7px 10px; font: inherit; font-size: 13px; border-radius: 8px; border: 1px solid color-mix(in srgb, CanvasText 10%, transparent); background: color-mix(in srgb, Canvas 70%, CanvasText 2%); color: CanvasText; outline: none; transition: border-color 0.2s, box-shadow 0.2s; }
.space-form__input:focus { border-color: color-mix(in srgb, Highlight 50%, CanvasText 20%); box-shadow: 0 0 0 3px color-mix(in srgb, Highlight 15%, transparent); }
.space-form__actions { display: flex; gap: 6px; }

.btn--close { appearance: none; border: none; background: transparent; cursor: pointer; display: flex; align-items: center; justify-content: center; padding: 4px; border-radius: 6px; color: color-mix(in srgb, CanvasText 55%, transparent); }
.btn--close:hover { background: color-mix(in srgb, CanvasText 6%, transparent); color: CanvasText; }

/* ===== Toolbar / search / existing styles ===== */
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

/* ===== Select mode toolbar ===== */
.select-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 14px; padding: 10px 14px; border-radius: 10px; border: 1px solid color-mix(in srgb, Highlight 30%, transparent); background: color-mix(in srgb, Highlight 8%, Canvas); }
.select-toolbar__info { display: flex; align-items: center; gap: 14px; min-width: 0; }
.select-toolbar__count { font-size: 13px; font-weight: 600; color: CanvasText; }
.select-toolbar__link { appearance: none; border: none; background: transparent; cursor: pointer; font: inherit; font-size: 12px; font-weight: 500; color: Highlight; padding: 2px 6px; border-radius: 4px; transition: background 0.15s; }
.select-toolbar__link:hover { background: color-mix(in srgb, Highlight 12%, transparent); }
.select-toolbar__actions { display: flex; gap: 6px; flex-shrink: 0; }

.entries-status { font-size: 12px; color: color-mix(in srgb, CanvasText 45%, transparent); margin-bottom: 10px; padding: 0 2px; }
.entries-status strong { font-weight: 700; color: color-mix(in srgb, CanvasText 65%, transparent); }
.entries-status__space { font-weight: 600; color: color-mix(in srgb, CanvasText 70%, transparent); }
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
