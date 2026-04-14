<script setup>
import { ref, computed, provide, onMounted, onUnmounted, nextTick } from 'vue'
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
} from '../wailsjs/go/app/App'
import { WindowSetBackgroundColour } from '../wailsjs/runtime/runtime'

import VaultTab from './components/VaultTab.vue'
import SyncTab from './components/SyncTab.vue'
import SettingsTab from './components/SettingsTab.vue'
import PasswordDialog from './components/PasswordDialog.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import ToastNotification from './components/ToastNotification.vue'

/* ---------- Window chrome ---------- */
function applyWindowChrome() {
  try {
    const dark = window.matchMedia('(prefers-color-scheme: dark)').matches
    WindowSetBackgroundColour(dark ? 24 : 250, dark ? 24 : 250, dark ? 27 : 252, 255)
  } catch { /* non-Wails env */ }
}

/* ---------- Tab / Config state ---------- */
const activeTab = ref('vault')
const appConfig = ref({
  config_path: '', repo_root: '', remote_url: '',
  vault_file_name: 'vault.dat', load_error: '',
})
const status = ref(null)
const syncing = ref(false)
const vaultUnlocked = ref(false)
const vaultTabRef = ref(null)

const vaultLabel = computed(() => appConfig.value.vault_file_name || 'vault.dat')

/* ---------- Toast ---------- */
const toastVisible = ref(false)
const toastMsg = ref('')
const toastType = ref('error')
let toastTimer = null

const showToast = (msg, type = 'error') => {
  toastMsg.value = msg
  toastType.value = type
  toastVisible.value = true
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toastVisible.value = false }, 4000)
}

const showErr = (e) => {
  showToast(typeof e === 'string' ? e : e?.message ?? String(e), 'error')
}
const showSuccess = (msg) => showToast(msg, 'success')

/* ---------- Password dialog ---------- */
const dlgPwdOpen = ref(false)
const dlgPwdTitle = ref('')
let dlgPwdResolve = null

const askPassword = (title) => {
  return new Promise((resolve) => {
    dlgPwdTitle.value = title
    dlgPwdResolve = resolve
    dlgPwdOpen.value = true
  })
}

const onPwdConfirm = (value) => {
  dlgPwdOpen.value = false
  const r = dlgPwdResolve; dlgPwdResolve = null
  if (r) r(value)
}
const onPwdCancel = () => {
  dlgPwdOpen.value = false
  const r = dlgPwdResolve; dlgPwdResolve = null
  if (r) r(null)
}

/* ---------- Confirm dialog ---------- */
const dlgConfirmOpen = ref(false)
const dlgConfirmMsg = ref('')
let dlgConfirmResolve = null

const askConfirm = (message) => {
  return new Promise((resolve) => {
    dlgConfirmMsg.value = message
    dlgConfirmResolve = resolve
    dlgConfirmOpen.value = true
  })
}

const onConfirmYes = () => {
  dlgConfirmOpen.value = false
  const r = dlgConfirmResolve; dlgConfirmResolve = null
  if (r) r(true)
}
const onConfirmNo = () => {
  dlgConfirmOpen.value = false
  const r = dlgConfirmResolve; dlgConfirmResolve = null
  if (r) r(false)
}

/* ---------- Provide utilities to children ---------- */
provide('askPassword', askPassword)
provide('askConfirm', askConfirm)
provide('showErr', showErr)
provide('showSuccess', showSuccess)

/* ---------- Config helpers ---------- */
const refreshAppConfig = async () => { appConfig.value = await GetAppConfig() }

const loadStatus = async () => {
  try { status.value = await GetRepoStatus() }
  catch (e) { status.value = null; showErr(e) }
}

const doReloadConfig = async () => {
  vaultTabRef.value?.lockVault()
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

/* ---------- Sync operations ---------- */
const doPull = async () => {
  syncing.value = true
  try {
    await Pull()
    await loadStatus()
    if (vaultUnlocked.value) await vaultTabRef.value?.refreshEntries()
    showSuccess('Pull 完成')
  } catch (e) { showErr(e) }
  finally { syncing.value = false }
}

const doPush = async () => {
  syncing.value = true
  try { await Push(); await loadStatus(); showSuccess('Push 完成') }
  catch (e) { showErr(e) }
  finally { syncing.value = false }
}

const doSync = async () => {
  const password = await askPassword(`输入主密码（用于解密 / 加密 ${vaultLabel.value}）`)
  if (password == null) return
  syncing.value = true
  try {
    await Sync(password)
    await loadStatus()
    if (vaultUnlocked.value) {
      await vaultTabRef.value?.refreshWithPassword(password)
    }
    showSuccess('同步完成')
  } catch (e) { showErr(e) }
  finally { syncing.value = false }
}

const doBindRepo = async () => {
  if (!appConfig.value.remote_url?.trim()) {
    showErr('请先在 pwdmgr.config.json 中填写 remote_url'); return
  }
  const password = await askPassword('输入主密码（新建或合并加密库时使用）')
  if (password == null) return
  syncing.value = true
  try {
    await BindRepo(password)
    await loadStatus()
    if (vaultUnlocked.value) {
      await vaultTabRef.value?.refreshWithPassword(password)
    }
    showSuccess('仓库绑定完成')
  } catch (e) { showErr(e) }
  finally { syncing.value = false }
}

const doInitLocalVault = async () => {
  const password = await askPassword(`设置主密码（用于加密 ${vaultLabel.value}）`)
  if (password == null) return
  try {
    await InitLocalVault(password)
    await loadStatus()
    showSuccess('本地加密库已创建')
  } catch (e) { showErr(e) }
}

const lockVault = () => { vaultTabRef.value?.lockVault() }

/* ---------- Lifecycle ---------- */
onMounted(async () => {
  applyWindowChrome()
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  const onSchemeChange = () => applyWindowChrome()
  mq.addEventListener('change', onSchemeChange)
  onUnmounted(() => mq.removeEventListener('change', onSchemeChange))

  await refreshAppConfig()
  if (!appConfig.value.load_error) await loadStatus()
})
</script>

<template>
  <div class="app">
    <!-- Top bar -->
    <header class="topbar">
      <div class="topbar__left">
        <svg class="topbar__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        <span class="topbar__title">kPass</span>
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
      <button :class="['tabs__btn', activeTab === 'vault' && 'tabs__btn--active']" @click="activeTab = 'vault'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
        保险库
      </button>
      <button :class="['tabs__btn', activeTab === 'sync' && 'tabs__btn--active']" @click="activeTab = 'sync'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        同步
        <span v-if="syncing" class="tabs__spinner" />
      </button>
      <button :class="['tabs__btn', activeTab === 'settings' && 'tabs__btn--active']" @click="activeTab = 'settings'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        设置
      </button>
    </nav>

    <!-- Main content -->
    <main class="content">
      <VaultTab
        v-show="activeTab === 'vault'"
        ref="vaultTabRef"
        :vault-label="vaultLabel"
        @update:unlocked="vaultUnlocked = $event"
      />
      <SyncTab
        v-show="activeTab === 'sync'"
        :syncing="syncing"
        :status="status"
        :app-config="appConfig"
        @sync="doSync"
        @pull="doPull"
        @push="doPush"
        @bind="doBindRepo"
        @init-local="doInitLocalVault"
        @refresh-status="loadStatus"
      />
      <SettingsTab
        v-show="activeTab === 'settings'"
        :app-config="appConfig"
        @reload="doReloadConfig"
      />
    </main>

    <!-- Toast -->
    <ToastNotification
      :visible="toastVisible"
      :message="toastMsg"
      :type="toastType"
      @close="toastVisible = false"
    />

    <!-- Dialogs -->
    <PasswordDialog
      :open="dlgPwdOpen"
      :title="dlgPwdTitle"
      @confirm="onPwdConfirm"
      @cancel="onPwdCancel"
    />
    <ConfirmDialog
      :open="dlgConfirmOpen"
      :message="dlgConfirmMsg"
      @confirm="onConfirmYes"
      @cancel="onConfirmNo"
    />
  </div>
</template>

<style scoped>
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

.topbar__left { display: flex; align-items: center; gap: 10px; }
.topbar__icon { width: 22px; height: 22px; color: Highlight; }
.topbar__title { font-size: 16px; font-weight: 700; letter-spacing: -0.02em; }
.topbar__badge { padding: 2px 8px; border-radius: 6px; font-size: 10px; font-weight: 600; letter-spacing: 0.04em; color: color-mix(in srgb, CanvasText 55%, transparent); background: color-mix(in srgb, CanvasText 6%, transparent); }
.topbar__right { display: flex; align-items: center; gap: 8px; }

.status-chip { display: inline-flex; align-items: center; gap: 5px; padding: 4px 10px; border-radius: 999px; font-size: 12px; font-weight: 600; cursor: default; transition: all 0.2s ease; }
.status-chip--unlocked { color: #16a34a; background: color-mix(in srgb, #16a34a 12%, transparent); cursor: pointer; }
.status-chip--unlocked:hover { background: color-mix(in srgb, #16a34a 18%, transparent); }
.status-chip--locked { color: color-mix(in srgb, CanvasText 55%, transparent); background: color-mix(in srgb, CanvasText 6%, transparent); }

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

.tabs__btn:hover { color: color-mix(in srgb, CanvasText 80%, transparent); background: color-mix(in srgb, CanvasText 3%, transparent); }
.tabs__btn--active { color: Highlight; border-bottom-color: Highlight; font-weight: 600; }

.tabs__spinner {
  width: 12px;
  height: 12px;
  border: 2px solid color-mix(in srgb, Highlight 30%, transparent);
  border-top-color: Highlight;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* ===== Content ===== */
.content {
  flex: 1;
  padding: 24px 20px 40px;
  max-width: 720px;
  width: 100%;
  margin: 0 auto;
}
</style>
