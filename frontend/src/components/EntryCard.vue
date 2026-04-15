<script setup>
import { ref } from 'vue'

defineProps({
  entry: Object,
  pwdVisible: Boolean,
  selectedTags: Array,
})

const emit = defineEmits(['toggle-pwd', 'copy', 'edit', 'delete', 'toggle-tag'])

const noteExpanded = ref(false)

const avatarColors = [
  ['#3b82f6', '#6366f1'],
  ['#8b5cf6', '#a855f7'],
  ['#ec4899', '#f43f5e'],
  ['#f97316', '#eab308'],
  ['#10b981', '#14b8a6'],
  ['#06b6d4', '#3b82f6'],
]

function getAvatarGradient(name) {
  const code = (name || '?').charCodeAt(0)
  const pair = avatarColors[code % avatarColors.length]
  return `linear-gradient(135deg, ${pair[0]}, ${pair[1]})`
}
</script>

<template>
  <div class="entry-card">
    <div class="entry-card__header">
      <div class="entry-card__info">
        <div class="entry-card__avatar" :style="{ background: getAvatarGradient(entry.name) }">
          {{ (entry.name || '?')[0].toUpperCase() }}
        </div>
        <div class="entry-card__meta">
          <span class="entry-card__name">{{ entry.name }}</span>
          <span class="entry-card__user">{{ entry.username || '—' }}</span>
        </div>
      </div>
      <div class="entry-card__tags" v-if="entry.tags && entry.tags.length">
        <button
          v-for="t in entry.tags"
          :key="t"
          :class="['tag', selectedTags.includes(t) && 'tag--active']"
          type="button"
          @click.stop="emit('toggle-tag', t)"
          :title="`按标签「${t}」筛选`"
        >{{ t }}</button>
      </div>
    </div>

    <div class="entry-card__pwd" @click="emit('copy', entry.password)" title="点击复制密码">
      <code class="entry-card__pwd-text">{{ pwdVisible ? entry.password : '••••••••••' }}</code>
      <div class="entry-card__pwd-actions">
        <button type="button" class="entry-card__pwd-btn" @click.stop="emit('toggle-pwd')" :title="pwdVisible ? '隐藏密码' : '显示密码'">
          <svg v-if="!pwdVisible" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
        </button>
        <button type="button" class="entry-card__pwd-btn" @click.stop="emit('copy', entry.password)" title="复制密码">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </button>
        <button type="button" class="entry-card__pwd-btn" @click.stop="emit('copy', entry.username)" title="复制用户名">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        </button>
      </div>
    </div>

    <Transition name="slide">
      <div v-if="noteExpanded && entry.note" class="entry-card__note">
        <div class="entry-card__note-content">{{ entry.note }}</div>
      </div>
    </Transition>

    <div class="entry-card__footer">
      <button type="button" class="entry-card__action" @click="emit('edit')">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        编辑
      </button>
      <button v-if="entry.note" type="button" :class="['entry-card__action', noteExpanded && 'entry-card__action--active']" @click="noteExpanded = !noteExpanded">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
        {{ noteExpanded ? '收起备注' : '备注' }}
      </button>
      <button type="button" class="entry-card__action entry-card__action--danger" @click="emit('delete')">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
        删除
      </button>
    </div>
  </div>
</template>

<style scoped>
.entry-card {
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  padding: 14px 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

.entry-card:hover {
  border-color: rgba(0, 0, 0, 0.12);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.07), 0 1px 3px rgba(0, 0, 0, 0.04);
}

@media (prefers-color-scheme: dark) {
  .entry-card {
    background: #1f1f23;
    border-color: rgba(255, 255, 255, 0.06);
  }
  .entry-card:hover {
    border-color: rgba(255, 255, 255, 0.12);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), 0 0 1px rgba(255, 255, 255, 0.05);
  }
}

/* --- Header --- */
.entry-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.entry-card__info {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.entry-card__avatar {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
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
  color: inherit;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-card__user {
  font-size: 12px;
  color: #8b8b8b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

@media (prefers-color-scheme: dark) {
  .entry-card__user { color: #6b6b6b; }
}

/* --- Tags --- */
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
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  background: color-mix(in srgb, Highlight 10%, transparent);
  color: Highlight;
  transition: all 0.12s ease;
}

.tag:hover { background: color-mix(in srgb, Highlight 18%, transparent); }
.tag--active { background: Highlight; color: #fff; }
.tag--active:hover { filter: brightness(1.08); }

/* --- Password row --- */
.entry-card__pwd {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 10px -16px;
  padding: 8px 16px;
  background: rgba(0, 0, 0, 0.02);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  cursor: pointer;
  transition: background 0.12s;
}

.entry-card__pwd:hover {
  background: rgba(0, 0, 0, 0.04);
}

@media (prefers-color-scheme: dark) {
  .entry-card__pwd {
    background: rgba(255, 255, 255, 0.02);
    border-top-color: rgba(255, 255, 255, 0.04);
    border-bottom-color: rgba(255, 255, 255, 0.04);
  }
  .entry-card__pwd:hover { background: rgba(255, 255, 255, 0.04); }
}

.entry-card__pwd-text {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  letter-spacing: 0.05em;
  color: #999;
  user-select: all;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (prefers-color-scheme: dark) {
  .entry-card__pwd-text { color: #666; }
}

.entry-card__pwd-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.entry-card:hover .entry-card__pwd-actions { opacity: 1; }

.entry-card__pwd-btn {
  appearance: none;
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 5px;
  border-radius: 5px;
  color: #999;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.12s ease;
}

.entry-card__pwd-btn:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #555;
}

@media (prefers-color-scheme: dark) {
  .entry-card__pwd-btn { color: #666; }
  .entry-card__pwd-btn:hover { background: rgba(255, 255, 255, 0.08); color: #aaa; }
}

/* --- Note --- */
.entry-card__note {
  margin: 0 -16px;
  padding: 10px 16px;
  background: rgba(0, 0, 0, 0.02);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.entry-card__note-content {
  font-size: 13px;
  line-height: 1.6;
  color: color-mix(in srgb, CanvasText 70%, transparent);
  white-space: pre-wrap;
  word-break: break-word;
}

@media (prefers-color-scheme: dark) {
  .entry-card__note {
    background: rgba(255, 255, 255, 0.02);
    border-top-color: rgba(255, 255, 255, 0.04);
    border-bottom-color: rgba(255, 255, 255, 0.04);
  }
}

/* --- Footer --- */
.entry-card__footer {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.entry-card:hover .entry-card__footer { opacity: 1; }

.entry-card__action {
  appearance: none;
  border: none;
  cursor: pointer;
  font: inherit;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 5px;
  font-size: 12px;
  font-weight: 500;
  color: #666;
  background: rgba(0, 0, 0, 0.04);
  transition: all 0.12s ease;
}

.entry-card__action:hover {
  background: rgba(0, 0, 0, 0.08);
  color: #333;
}

.entry-card__action--active {
  background: color-mix(in srgb, Highlight 12%, transparent);
  color: Highlight;
}

.entry-card__action--active:hover {
  background: color-mix(in srgb, Highlight 18%, transparent);
}

.entry-card__action--danger:hover {
  background: rgba(220, 38, 38, 0.08);
  color: #dc2626;
}

@media (prefers-color-scheme: dark) {
  .entry-card__action { color: #888; background: rgba(255, 255, 255, 0.05); }
  .entry-card__action:hover { background: rgba(255, 255, 255, 0.1); color: #ccc; }
  .entry-card__action--danger:hover { background: rgba(220, 38, 38, 0.15); color: #f87171; }
}

/* --- Mobile --- */
@media (max-width: 520px) {
  .entry-card__header { flex-direction: column; align-items: flex-start; }
  .entry-card__pwd-actions { opacity: 1; }
  .entry-card__footer { opacity: 1; }
}
</style>
