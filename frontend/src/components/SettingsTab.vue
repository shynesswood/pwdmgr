<script setup>
defineProps({
  appConfig: Object,
})

const emit = defineEmits(['reload'])
</script>

<template>
  <div class="page">
    <div class="section-header">
      <h2 class="section-header__title">应用配置</h2>
      <p class="section-header__desc">
        仓库路径与远程地址由工作目录下的 <code>pwdmgr.config.json</code> 提供。
        环境变量 <code>PWDMGR_CONFIG</code> 可指向任意路径的配置文件。
      </p>
    </div>

    <div v-if="appConfig.load_error" class="alert" role="alert">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      {{ appConfig.load_error }}
    </div>

    <div class="config-list">
      <div class="config-item">
        <div class="config-item__label">配置文件</div>
        <div class="config-item__value">{{ appConfig.config_path || '—' }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">仓库根目录</div>
        <div class="config-item__value">{{ appConfig.repo_root || '—' }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">远程 URL</div>
        <div class="config-item__value">{{ appConfig.remote_url || '（未填写，仅本地操作）' }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">加密库文件名</div>
        <div class="config-item__value"><code>{{ appConfig.vault_file_name }}</code></div>
      </div>
    </div>

    <div style="margin-top: 20px;">
      <button type="button" class="btn btn--ghost" @click="emit('reload')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        重新加载配置
      </button>
    </div>
  </div>
</template>

<style scoped>
.config-list {
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  overflow: hidden;
}

.config-item {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid color-mix(in srgb, CanvasText 5%, transparent);
  transition: background 0.15s;
}

.config-item:last-child { border-bottom: none; }
.config-item:hover { background: color-mix(in srgb, CanvasText 2%, transparent); }

.config-item__label {
  font-size: 12px;
  font-weight: 600;
  color: color-mix(in srgb, CanvasText 60%, transparent);
  padding-top: 1px;
}

.config-item__value {
  font-size: 13px;
  line-height: 1.5;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.config-item__value code {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
  background: color-mix(in srgb, CanvasText 7%, transparent);
}

.alert {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 20px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.5;
  color: #dc2626;
  background: color-mix(in srgb, #dc2626 8%, Canvas);
  border: 1px solid color-mix(in srgb, #dc2626 20%, transparent);
}

.alert svg { flex-shrink: 0; margin-top: 1px; }

@media (max-width: 520px) {
  .config-item { grid-template-columns: 1fr; gap: 4px; }
}
</style>
