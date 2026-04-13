<script setup>
import { computed } from 'vue'

const props = defineProps({
  syncing: Boolean,
  status: Object,
  appConfig: Object,
})

const emit = defineEmits(['sync', 'pull', 'push', 'bind', 'init-local', 'refresh-status'])

const hasRemoteUrl = computed(() => !!props.appConfig?.remote_url?.trim())
const isGitRepo = computed(() => !!props.status?.isGitRepo)
const hasLocalVault = computed(() => !!props.status?.hasLocalVault)
const hasRemote = computed(() => !!props.status?.hasRemote)

const setupState = computed(() => {
  if (!props.status) return 'unknown'
  if (hasLocalVault.value && hasRemote.value) return 'ready'
  if (hasLocalVault.value && !hasRemoteUrl.value) return 'local-only'
  if (hasLocalVault.value && hasRemoteUrl.value && !hasRemote.value) return 'need-bind'
  return 'need-init'
})

const setupHint = computed(() => {
  switch (setupState.value) {
    case 'ready': return { icon: 'check', type: 'ok', text: '本地库与远程仓库已就绪，可正常使用上方的同步功能。' }
    case 'local-only': return { icon: 'info', type: 'info', text: '本地库已创建（仅本地）。如需多设备同步，请在配置文件中填写 remote_url 后点击「绑定远程仓库」。' }
    case 'need-bind': return { icon: 'warn', type: 'warn', text: '本地库已存在但尚未绑定远程仓库，点击「绑定远程仓库」完成关联。' }
    case 'need-init': return { icon: 'info', type: 'info', text: '请选择一种方式初始化您的加密库。' }
    default: return null
  }
})

const pipelineSteps = computed(() => {
  const s = props.status
  if (!s) return []
  return [
    { key: 'git', label: 'Git 仓库', done: s.isGitRepo },
    { key: 'remote', label: '远程连接', done: s.hasRemote },
    { key: 'data', label: '远程数据', done: s.remoteHasData },
    { key: 'vault', label: '本地加密库', done: s.hasLocalVault },
  ]
})

const completedCount = computed(() => pipelineSteps.value.filter(s => s.done).length)

const overallDotClass = computed(() => {
  if (!props.status) return 'dot--idle'
  if (props.status.hasUncommitted) return 'dot--warn'
  if (completedCount.value === 4) return 'dot--ok'
  return 'dot--idle'
})

const infoItems = computed(() => {
  const s = props.status
  if (!s) return []
  const items = []
  if (s.currentBranch) {
    items.push({ label: '分支', value: s.currentBranch, icon: 'branch' })
  }
  if (s.remoteURL) {
    items.push({ label: '远程', value: s.remoteURL, icon: 'link' })
  }
  items.push({
    label: '工作区',
    value: !s.isGitRepo ? '未初始化' : s.hasUncommitted ? '有未提交更改' : '干净',
    icon: 'dir',
    warn: s.hasUncommitted,
  })
  return items
})
</script>

<template>
  <div class="page">
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
        <button type="button" class="btn btn--primary" :disabled="syncing" @click="emit('sync')">
          {{ syncing ? '同步中…' : 'Sync' }}
        </button>
      </div>

      <div class="sync-card">
        <div class="sync-card__icon sync-card__icon--green">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 20 1 14 7 14"/><path d="M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        </div>
        <h3 class="sync-card__title">Pull</h3>
        <p class="sync-card__desc">从远程仓库拉取最新更改到本地。</p>
        <button type="button" class="btn btn--ghost" :disabled="syncing" @click="emit('pull')">Pull</button>
      </div>

      <div class="sync-card">
        <div class="sync-card__icon sync-card__icon--orange">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M23 10l-4.64-4.36A9 9 0 0 0 3.51 9"/></svg>
        </div>
        <h3 class="sync-card__title">Push</h3>
        <p class="sync-card__desc">将本地提交推送到远程仓库。</p>
        <button type="button" class="btn btn--ghost" :disabled="syncing" @click="emit('push')">Push</button>
      </div>
    </div>

    <!-- Setup section -->
    <div class="section-header" style="margin-top: 28px;">
      <h2 class="section-header__title">仓库设置</h2>
    </div>

    <div v-if="setupHint" :class="['setup-hint', `setup-hint--${setupHint.type}`]">
      <svg v-if="setupHint.icon === 'check'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
      <svg v-else-if="setupHint.icon === 'warn'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
      <span>{{ setupHint.text }}</span>
    </div>

    <div class="init-grid">
      <div :class="['init-card', hasLocalVault && 'init-card--done']">
        <div class="init-card__icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        </div>
        <h3 class="init-card__title">创建本地库</h3>
        <p class="init-card__desc">首次使用时，在本地创建全新的加密库并初始化 Git 仓库。</p>
        <div v-if="hasLocalVault" class="init-card__badge init-card__badge--done">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          已创建
        </div>
        <button v-else type="button" class="btn btn--accent" :disabled="syncing || hasLocalVault" @click="emit('init-local')">创建</button>
      </div>

      <div :class="['init-card', (hasRemote && hasLocalVault) && 'init-card--done']">
        <div class="init-card__icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
        </div>
        <h3 class="init-card__title">绑定远程仓库</h3>
        <p class="init-card__desc">
          <template v-if="!hasRemoteUrl">请先在 <code>pwdmgr.config.json</code> 中配置 <code>remote_url</code>。</template>
          <template v-else>关联远程 Git 仓库（如 GitHub），自动处理首次同步与合并。</template>
        </p>
        <div v-if="hasRemote && hasLocalVault" class="init-card__badge init-card__badge--done">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          已绑定
        </div>
        <button v-else type="button" class="btn btn--accent" :disabled="syncing || !hasRemoteUrl" @click="emit('bind')">绑定并同步</button>
      </div>
    </div>

    <!-- Repo status -->
    <div class="status-block">
      <div class="status-block__head">
        <div class="status-block__left">
          <span :class="['dot', overallDotClass]" />
          <span class="status-block__title">仓库状态</span>
          <span v-if="status" class="status-block__count">{{ completedCount }}/4</span>
        </div>
        <button type="button" class="btn btn--ghost btn--sm" @click="emit('refresh-status')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          刷新
        </button>
      </div>

      <div class="status-block__body">
        <!-- Empty state -->
        <div v-if="!status" class="status-empty">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
          <span>配置有效时点击「刷新」获取状态</span>
        </div>

        <template v-else>
          <!-- Pipeline -->
          <div class="pipeline">
            <div v-for="(step, i) in pipelineSteps" :key="step.key" class="pipeline__step">
              <div :class="['pipeline__node', step.done && 'pipeline__node--done']">
                <svg v-if="step.done" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                <span v-else class="pipeline__num">{{ i + 1 }}</span>
              </div>
              <div v-if="i < pipelineSteps.length - 1" :class="['pipeline__line', step.done && pipelineSteps[i+1].done && 'pipeline__line--done']" />
              <span :class="['pipeline__label', step.done && 'pipeline__label--done']">{{ step.label }}</span>
            </div>
          </div>

          <!-- Info items -->
          <div v-if="infoItems.length" class="info-grid">
            <div v-for="item in infoItems" :key="item.label" class="info-item">
              <div class="info-item__icon">
                <!-- branch -->
                <svg v-if="item.icon === 'branch'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/></svg>
                <!-- link -->
                <svg v-else-if="item.icon === 'link'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                <!-- dir -->
                <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
              </div>
              <span class="info-item__label">{{ item.label }}</span>
              <span :class="['info-item__value', item.warn && 'info-item__value--warn']">{{ item.value }}</span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
  .sync-card { background: color-mix(in srgb, Canvas 80%, CanvasText 20%); }
}

.sync-card__icon {
  width: 44px; height: 44px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 12px;
}

.sync-card__icon--blue  { background: color-mix(in srgb, #3b82f6 12%, transparent); color: #3b82f6; }
.sync-card__icon--green { background: color-mix(in srgb, #16a34a 12%, transparent); color: #16a34a; }
.sync-card__icon--orange { background: color-mix(in srgb, #ea580c 12%, transparent); color: #ea580c; }

.sync-card__title { margin: 0; font-size: 15px; font-weight: 600; }
.sync-card__desc { margin: 0; font-size: 12px; color: color-mix(in srgb, CanvasText 55%, transparent); line-height: 1.5; flex: 1; }
.sync-card :deep(.btn) { width: 100%; justify-content: center; margin-top: 4px; }

/* ===== Setup hint ===== */
.setup-hint {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  margin-bottom: 16px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.5;
}

.setup-hint svg { flex-shrink: 0; margin-top: 1px; }
.setup-hint--ok   { color: #16a34a; background: color-mix(in srgb, #16a34a 8%, Canvas); border: 1px solid color-mix(in srgb, #16a34a 18%, transparent); }
.setup-hint--info { color: color-mix(in srgb, CanvasText 65%, Highlight 35%); background: color-mix(in srgb, Highlight 6%, Canvas); border: 1px solid color-mix(in srgb, Highlight 15%, transparent); }
.setup-hint--warn { color: #ea580c; background: color-mix(in srgb, #ea580c 8%, Canvas); border: 1px solid color-mix(in srgb, #ea580c 18%, transparent); }

/* ===== Init cards ===== */
.init-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 28px;
}

@media (max-width: 520px) {
  .init-grid { grid-template-columns: 1fr; }
}

.init-card {
  padding: 20px;
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.2s;
}

@media (prefers-color-scheme: dark) {
  .init-card { background: color-mix(in srgb, Canvas 80%, CanvasText 20%); }
}

.init-card--done { border-color: color-mix(in srgb, #16a34a 25%, transparent); }

.init-card__icon {
  width: 44px; height: 44px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 12px;
  background: color-mix(in srgb, Highlight 10%, transparent);
  color: Highlight;
}

.init-card--done .init-card__icon {
  background: color-mix(in srgb, #16a34a 10%, transparent);
  color: #16a34a;
}

.init-card__title { margin: 0; font-size: 15px; font-weight: 600; }

.init-card__desc {
  margin: 0;
  font-size: 12px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  line-height: 1.5;
  flex: 1;
}

.init-card__desc code {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 4px;
  background: color-mix(in srgb, CanvasText 7%, transparent);
}

.init-card__badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  width: fit-content;
}

.init-card__badge--done {
  color: #16a34a;
  background: color-mix(in srgb, #16a34a 10%, transparent);
}

.init-card .btn { width: 100%; justify-content: center; }

/* ===== Status block ===== */
.status-block {
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  overflow: hidden;
  background: color-mix(in srgb, Canvas 94%, CanvasText 6%);
}

@media (prefers-color-scheme: dark) {
  .status-block { background: color-mix(in srgb, Canvas 80%, CanvasText 20%); }
}

.status-block__head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
}

.status-block__left { display: flex; align-items: center; gap: 8px; }
.dot { width: 8px; height: 8px; border-radius: 50%; }
.dot--ok   { background: #16a34a; box-shadow: 0 0 6px color-mix(in srgb, #16a34a 40%, transparent); }
.dot--warn { background: #ea580c; box-shadow: 0 0 6px color-mix(in srgb, #ea580c 40%, transparent); }
.dot--idle { background: color-mix(in srgb, CanvasText 30%, transparent); }

.status-block__title { font-size: 13px; font-weight: 600; }

.status-block__count {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: 6px;
  background: color-mix(in srgb, CanvasText 6%, transparent);
  color: color-mix(in srgb, CanvasText 55%, transparent);
}

.status-block__body { padding: 20px 16px; }

/* Empty */
.status-empty {
  display: flex;
  align-items: center;
  gap: 10px;
  color: color-mix(in srgb, CanvasText 40%, transparent);
  font-size: 13px;
}

/* ===== Pipeline ===== */
.pipeline {
  display: flex;
  align-items: flex-start;
  gap: 0;
  margin-bottom: 20px;
}

.pipeline__step {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
}

.pipeline__node {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid color-mix(in srgb, CanvasText 18%, transparent);
  background: Canvas;
  color: color-mix(in srgb, CanvasText 40%, transparent);
  position: relative;
  z-index: 1;
  transition: all 0.25s ease;
}

.pipeline__node--done {
  border-color: #16a34a;
  background: #16a34a;
  color: #fff;
}

.pipeline__num {
  font-size: 12px;
  font-weight: 700;
}

.pipeline__line {
  position: absolute;
  top: 15px;
  left: calc(50% + 16px);
  right: calc(-50% + 16px);
  height: 2px;
  background: color-mix(in srgb, CanvasText 12%, transparent);
  z-index: 0;
  transition: background 0.25s ease;
}

.pipeline__line--done {
  background: #16a34a;
}

.pipeline__label {
  margin-top: 8px;
  font-size: 11px;
  font-weight: 500;
  color: color-mix(in srgb, CanvasText 45%, transparent);
  text-align: center;
  transition: color 0.25s ease;
}

.pipeline__label--done {
  color: color-mix(in srgb, CanvasText 75%, transparent);
  font-weight: 600;
}

/* ===== Info grid ===== */
.info-grid {
  display: flex;
  flex-direction: column;
  gap: 0;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, CanvasText 6%, transparent);
  overflow: hidden;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 5%, transparent);
  transition: background 0.15s;
}

.info-item:last-child { border-bottom: none; }
.info-item:hover { background: color-mix(in srgb, CanvasText 2%, transparent); }

.info-item__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: color-mix(in srgb, CanvasText 40%, transparent);
  flex-shrink: 0;
}

.info-item__label {
  font-size: 12px;
  font-weight: 600;
  color: color-mix(in srgb, CanvasText 50%, transparent);
  width: 48px;
  flex-shrink: 0;
}

.info-item__value {
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  color: color-mix(in srgb, CanvasText 75%, transparent);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-item__value--warn {
  color: #ea580c;
  font-weight: 600;
  font-family: inherit;
}

@media (max-width: 520px) {
  .pipeline__node { width: 28px; height: 28px; }
  .pipeline__node--done svg { width: 12px; height: 12px; }
  .pipeline__num { font-size: 11px; }
  .pipeline__line { top: 13px; left: calc(50% + 14px); right: calc(-50% + 14px); }
  .pipeline__label { font-size: 10px; }
}
</style>
