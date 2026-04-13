<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  open: Boolean,
  title: String,
})

const emit = defineEmits(['confirm', 'cancel'])

const inputValue = ref('')
const inputRef = ref(null)

watch(
  () => props.open,
  (val) => {
    if (val) {
      inputValue.value = ''
      nextTick(() => inputRef.value?.focus?.())
    }
  },
)

const confirm = () => {
  const v = inputValue.value.trim()
  emit('confirm', v === '' ? null : v)
}

const cancel = () => emit('cancel')
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" role="presentation" @click.self="cancel">
        <div class="modal" role="dialog" aria-modal="true" @keydown.escape.prevent="cancel">
          <div class="modal__icon">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/><circle cx="12" cy="16" r="1"/></svg>
          </div>
          <h3 class="modal__title">{{ title }}</h3>
          <input
            ref="inputRef"
            v-model="inputValue"
            type="password"
            class="field__input"
            placeholder="输入主密码"
            autocomplete="current-password"
            @keydown.enter.prevent="confirm"
            @keydown.escape.prevent="cancel"
          />
          <p class="modal__hint">留空并确定视为取消</p>
          <div class="modal__actions">
            <button type="button" class="btn btn--ghost" @click="cancel">取消</button>
            <button type="button" class="btn btn--primary" @click="confirm">确定</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
