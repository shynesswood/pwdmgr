<script setup>
defineProps({
  visible: Boolean,
  message: String,
  type: { type: String, default: 'error' },
})

const emit = defineEmits(['close'])
</script>

<template>
  <Transition name="toast">
    <div v-if="visible" :class="['toast', `toast--${type}`]" @click="emit('close')">
      <svg v-if="type === 'success'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <span class="toast__msg">{{ message }}</span>
    </div>
  </Transition>
</template>

<style scoped>
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
</style>
