<script setup>
defineProps({
  mode: { type: String, default: 'add' },
  form: { type: Object, required: true },
})
const emit = defineEmits(['submit', 'cancel'])
</script>

<template>
  <div class="form-panel">
    <div class="form-panel__head">
      <h3 class="form-panel__title">{{ mode === 'add' ? '新建条目' : '编辑条目' }}</h3>
      <button type="button" class="btn--close" @click="emit('cancel')">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <div class="form-panel__body">
      <div class="form-grid">
        <label class="field"><span class="field__label">名称</span><input v-model="form.name" class="field__input" type="text" placeholder="例如：GitHub" autocomplete="off" /></label>
        <label class="field"><span class="field__label">用户名</span><input v-model="form.username" class="field__input" type="text" placeholder="用户名或邮箱" autocomplete="off" /></label>
        <label class="field field--full"><span class="field__label">密码</span><input v-model="form.password" class="field__input field__input--mono" type="text" placeholder="密码" autocomplete="off" /></label>
        <label class="field field--full"><span class="field__label">标签</span><input v-model="form.tagsStr" class="field__input" type="text" placeholder="多个用逗号分隔，例如：工作, 社交" autocomplete="off" /></label>
        <label class="field field--full"><span class="field__label">备注</span><textarea v-model="form.note" class="field__input field__textarea" rows="2" placeholder="可选备注信息" /></label>
      </div>
      <div class="form-panel__actions">
        <button type="button" class="btn btn--ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="btn btn--primary" @click="emit('submit')">{{ mode === 'add' ? '添加' : '保存修改' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.form-panel { border-radius: 10px; border: 1px solid color-mix(in srgb, Highlight 20%, color-mix(in srgb, CanvasText 8%, transparent)); background: color-mix(in srgb, Highlight 3%, Canvas); overflow: hidden; }
.form-panel__head { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid color-mix(in srgb, CanvasText 5%, transparent); }
.form-panel__title { margin: 0; font-size: 13px; font-weight: 600; }
.form-panel__body { padding: 16px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.form-panel__actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 14px; padding-top: 12px; border-top: 1px solid color-mix(in srgb, CanvasText 5%, transparent); }

@media (max-width: 520px) {
  .form-grid { grid-template-columns: 1fr; }
}
</style>
