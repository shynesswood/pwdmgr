<script setup>
defineProps({
  open: Boolean,
  message: String,
})

const emit = defineEmits(['confirm', 'cancel'])
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" role="presentation" @click.self="emit('cancel')">
        <div class="modal modal--narrow" role="dialog" aria-modal="true" @keydown.escape.prevent="emit('cancel')">
          <div class="modal__icon modal__icon--warn">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          </div>
          <p class="modal__text">{{ message }}</p>
          <div class="modal__actions">
            <button type="button" class="btn btn--ghost" @click="emit('cancel')">取消</button>
            <button type="button" class="btn btn--danger" @click="emit('confirm')">确定</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
