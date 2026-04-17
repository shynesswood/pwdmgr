<script setup>
import { computed } from 'vue'

const props = defineProps({
  open: Boolean,
  title: { type: String, default: '选择目标空间' },
  description: { type: String, default: '' },
  spaces: { type: Array, default: () => [] },
  excludeId: { type: String, default: '' },
})

const emit = defineEmits(['confirm', 'cancel'])

const visibleSpaces = computed(() =>
  (props.spaces || []).filter((s) => s.id !== props.excludeId),
)
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" role="presentation" @click.self="emit('cancel')">
        <div class="modal modal--narrow" role="dialog" aria-modal="true" @keydown.escape.prevent="emit('cancel')">
          <h3 class="modal__title">{{ title }}</h3>
          <p v-if="description" class="modal__subtitle">{{ description }}</p>
          <div v-if="visibleSpaces.length === 0" class="picker__empty">
            没有其它空间可选，先新建一个空间吧。
          </div>
          <div v-else class="picker__list">
            <button
              v-for="s in visibleSpaces"
              :key="s.id"
              type="button"
              class="picker__item"
              @click="emit('confirm', s.id)"
            >
              <svg v-if="s.id === 'default'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <rect x="3" y="7" width="18" height="14" rx="2"/>
                <path d="M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
              <span class="picker__name">{{ s.name }}</span>
              <svg class="picker__chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
            </button>
          </div>
          <div class="modal__actions">
            <button type="button" class="btn btn--ghost" @click="emit('cancel')">取消</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal__title { margin: 0 0 4px; font-size: 15px; font-weight: 700; letter-spacing: -0.01em; }
.modal__subtitle { margin: 0 0 14px; font-size: 12px; color: color-mix(in srgb, CanvasText 55%, transparent); }

.picker__empty { padding: 20px 8px; font-size: 13px; text-align: center; color: color-mix(in srgb, CanvasText 50%, transparent); }

.picker__list { display: flex; flex-direction: column; gap: 4px; max-height: 320px; overflow-y: auto; margin: 4px 0 14px; padding-right: 2px; }

.picker__item { appearance: none; border: 1px solid color-mix(in srgb, CanvasText 8%, transparent); background: color-mix(in srgb, Canvas 92%, CanvasText 4%); cursor: pointer; font: inherit; display: flex; align-items: center; gap: 10px; padding: 10px 12px; border-radius: 8px; color: CanvasText; transition: all 0.12s ease; text-align: left; }
.picker__item:hover { border-color: color-mix(in srgb, Highlight 40%, CanvasText 10%); background: color-mix(in srgb, Highlight 6%, Canvas); }
.picker__item:hover .picker__chevron { color: Highlight; transform: translateX(2px); }

.picker__name { flex: 1; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.picker__chevron { color: color-mix(in srgb, CanvasText 45%, transparent); transition: all 0.15s ease; flex-shrink: 0; }
</style>
