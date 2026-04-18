<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  appConfig: Object,
  // onSave(form) -> Promise<any>：父组件负责调用后端接口并抛出异常（子组件据此停留在编辑态）
  onSave: { type: Function, default: null },
})

const emit = defineEmits(['reload'])

const editing = ref(false)
const saving = ref(false)

const form = ref({
  repo_root: '',
  remote_url: '',
  git_client: 'exec',
})

const resetFormFromConfig = () => {
  form.value = {
    repo_root: props.appConfig?.repo_root || '',
    remote_url: props.appConfig?.remote_url || '',
    git_client: props.appConfig?.git_client || 'exec',
  }
}

watch(
  () => [props.appConfig?.repo_root, props.appConfig?.remote_url, props.appConfig?.git_client],
  () => { if (!editing.value) resetFormFromConfig() },
  { immediate: true },
)

const gitClientLabel = computed(() => {
  const v = props.appConfig?.git_client || 'exec'
  if (v === 'go-git') return 'go-git（纯 Go 实现，不依赖本机 git）'
  return 'exec（调用本机 git 命令）'
})

const startEdit = () => {
  resetFormFromConfig()
  editing.value = true
}

const cancelEdit = () => {
  editing.value = false
  resetFormFromConfig()
}

const submit = async () => {
  if (saving.value) return
  if (!props.onSave) { editing.value = false; return }
  saving.value = true
  try {
    await props.onSave({ ...form.value })
    editing.value = false
  } catch {
    // 父组件已经吐出 toast；保持编辑态让用户修正
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="section-header">
      <h2 class="section-header__title">应用配置</h2>
      <p class="section-header__desc">
        仓库路径、远程地址与 Git 客户端可在下方编辑，保存后会写回
        <code>pwdmgr.config.json</code>。也可通过环境变量
        <code>PWDMGR_CONFIG</code> 指定任意路径的配置文件。
      </p>
    </div>

    <div v-if="appConfig.load_error" class="alert" role="alert">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      {{ appConfig.load_error }}
    </div>

    <!-- 只读模式 -->
    <div v-if="!editing" class="config-list">
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
        <div class="config-item__label">Git 客户端</div>
        <div class="config-item__value">{{ gitClientLabel }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">加密库文件名</div>
        <div class="config-item__value"><code>{{ appConfig.vault_file_name }}</code></div>
      </div>
    </div>

    <!-- 编辑模式 -->
    <form v-else class="config-form" @submit.prevent="submit">
      <div class="config-form__item">
        <label class="config-form__label" for="cfg-repo-root">仓库根目录 <span class="required">*</span></label>
        <input
          id="cfg-repo-root"
          v-model="form.repo_root"
          class="input"
          type="text"
          placeholder="/绝对路径/到本地 Git 仓库目录"
          autocomplete="off"
          spellcheck="false"
          required
        />
        <div class="hint">本地 Git 仓库的绝对路径，密码库文件 <code>{{ appConfig.vault_file_name }}</code> 会存放于此。</div>
      </div>

      <div class="config-form__item">
        <label class="config-form__label" for="cfg-remote-url">远程 URL</label>
        <input
          id="cfg-remote-url"
          v-model="form.remote_url"
          class="input"
          type="text"
          placeholder="git@github.com:用户/仓库.git（可留空）"
          autocomplete="off"
          spellcheck="false"
        />
        <div class="hint">留空则仅本地使用，不同步。建议使用 SSH key 或 token 认证。</div>
      </div>

      <div class="config-form__item">
        <label class="config-form__label" for="cfg-git-client">Git 客户端</label>
        <select id="cfg-git-client" v-model="form.git_client" class="input">
          <option value="exec">exec — 调用本机 git 命令（默认）</option>
          <option value="go-git">go-git — 纯 Go 实现，不依赖本机 git</option>
        </select>
        <div class="hint">
          go-git 模式下 Pull 仅支持 fast-forward / 合并（不支持 <code>--rebase</code>）；若你习惯手动改动仓库文件，推荐保持默认 exec。
        </div>
      </div>

      <div class="config-form__actions">
        <button type="button" class="btn btn--ghost" :disabled="saving" @click="cancelEdit">取消</button>
        <button type="submit" class="btn btn--primary" :disabled="saving">
          <svg v-if="saving" class="spinner" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>

    <div v-if="!editing" class="toolbar">
      <button type="button" class="btn btn--primary" @click="startEdit">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 1 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
        编辑配置
      </button>
      <button type="button" class="btn btn--ghost" @click="emit('reload')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        从磁盘重新加载
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

.config-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 18px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  background: color-mix(in srgb, Canvas 92%, transparent);
}

.config-form__item { display: flex; flex-direction: column; gap: 6px; }

.config-form__label {
  font-size: 12px;
  font-weight: 600;
  color: color-mix(in srgb, CanvasText 70%, transparent);
}

.config-form__label .required { color: #dc2626; margin-left: 2px; }

.input {
  appearance: none;
  width: 100%;
  padding: 9px 12px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, CanvasText 15%, transparent);
  background: Canvas;
  color: CanvasText;
  font: inherit;
  font-size: 13px;
  line-height: 1.4;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
  box-sizing: border-box;
}

.input:focus {
  border-color: Highlight;
  box-shadow: 0 0 0 3px color-mix(in srgb, Highlight 20%, transparent);
}

select.input {
  padding-right: 30px;
  background-image: linear-gradient(45deg, transparent 50%, currentColor 50%), linear-gradient(-45deg, transparent 50%, currentColor 50%);
  background-position: calc(100% - 16px) 50%, calc(100% - 11px) 50%;
  background-size: 5px 5px;
  background-repeat: no-repeat;
}

.hint {
  font-size: 12px;
  color: color-mix(in srgb, CanvasText 55%, transparent);
  line-height: 1.5;
}

.hint code {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  background: color-mix(in srgb, CanvasText 7%, transparent);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.config-form__actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding-top: 6px;
}

.toolbar {
  display: flex;
  gap: 8px;
  margin-top: 20px;
}

.spinner {
  animation: spin 0.8s linear infinite;
  margin-right: 2px;
  vertical-align: -2px;
}

@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 520px) {
  .config-item { grid-template-columns: 1fr; gap: 4px; }
  .toolbar { flex-direction: column; align-items: stretch; }
}
</style>
