<script setup>
import { ref, computed, watch, inject } from 'vue'

const askConfirm = inject('askConfirm', async () => true)

const props = defineProps({
  appConfig: Object,
  // onSave(form) -> Promise<any>：父组件负责调用后端接口并抛出异常（子组件据此停留在编辑态）
  onSave: { type: Function, default: null },
  // onSaveSsh({ ssh_key_path, ssh_key_passphrase }) -> Promise<snapshot>
  // 空串表示清除。仅当前端 saveSSH / clearSSH 触发时调用。
  // 注意 prop 名避开连续大写，便于生成 on-save-ssh 这种干净的 kebab-case 形式。
  onSaveSsh: { type: Function, default: null },
})

const emit = defineEmits(['reload'])

const editing = ref(false)
const saving = ref(false)

const form = ref({
  repo_root: '',
  remote_url: '',
  git_client: 'exec',
})

// SSH 凭据独立的一小块表单
const sshForm = ref({
  ssh_key_path: '',
  ssh_key_passphrase: '',
})
const sshSaving = ref(false)
const showPass = ref(false)

const resetFormFromConfig = () => {
  form.value = {
    repo_root: props.appConfig?.repo_root || '',
    remote_url: props.appConfig?.remote_url || '',
    git_client: props.appConfig?.git_client || 'exec',
  }
}

// 进入编辑态时，ssh_key_path 用当前值回显；口令不回显（后端根本不返回明文）
const resetSSHFormFromConfig = () => {
  sshForm.value = {
    ssh_key_path: props.appConfig?.ssh_key_path || '',
    ssh_key_passphrase: '',
  }
  showPass.value = false
}

watch(
  () => [props.appConfig?.repo_root, props.appConfig?.remote_url, props.appConfig?.git_client],
  () => { if (!editing.value) resetFormFromConfig() },
  { immediate: true },
)

watch(
  () => [props.appConfig?.ssh_key_path, props.appConfig?.ssh_key_has_pass],
  () => { if (!editing.value) resetSSHFormFromConfig() },
  { immediate: true },
)

const gitClientLabel = computed(() => {
  const v = props.appConfig?.git_client || 'exec'
  if (v === 'go-git') return 'go-git（纯 Go 实现，不依赖本机 git）'
  return 'exec（调用本机 git 命令）'
})

const sshPathLabel = computed(() => {
  const p = props.appConfig?.ssh_key_path
  return p ? p : '（未指定，自动探测 ssh-agent / ~/.ssh/id_*）'
})

const sshPassLabel = computed(() => props.appConfig?.ssh_key_has_pass ? '已设置（加密私钥）' : '未设置（私钥未加密）')

// 仅 go-git 模式下 SSH 凭据才实际生效；exec 模式给个温和的提示，不禁用输入
const sshOnlyForGoGit = computed(() => (props.appConfig?.git_client || 'exec') !== 'go-git')

const startEdit = () => {
  resetFormFromConfig()
  resetSSHFormFromConfig()
  editing.value = true
}

const cancelEdit = () => {
  editing.value = false
  resetFormFromConfig()
  resetSSHFormFromConfig()
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

const saveSSH = async () => {
  if (sshSaving.value || !props.onSaveSsh) return

  // 后端语义是"完整覆盖"：path 非空、口令留空 = 设为未加密。如果后端原本保存过口令，
  // 这个操作会清除它 —— 先让用户确认，避免误清除。
  const pathFilled = (sshForm.value.ssh_key_path || '').trim() !== ''
  const passEmpty = (sshForm.value.ssh_key_passphrase || '') === ''
  const hadPass = !!props.appConfig?.ssh_key_has_pass
  if (pathFilled && passEmpty && hadPass) {
    const ok = await askConfirm(
      '当前已保存了私钥口令，但口令框留空 —— 继续保存会把口令清除（等同"切换为未加密私钥"）。是否继续？'
    )
    if (!ok) return
  }

  sshSaving.value = true
  try {
    await props.onSaveSsh({
      ssh_key_path: sshForm.value.ssh_key_path,
      ssh_key_passphrase: sshForm.value.ssh_key_passphrase,
    })
    sshForm.value.ssh_key_passphrase = ''
    showPass.value = false
  } catch {
    /* 父组件已 toast；保持表单让用户修正 */
  } finally {
    sshSaving.value = false
  }
}

const clearSSH = async () => {
  if (sshSaving.value || !props.onSaveSsh) return
  sshSaving.value = true
  try {
    await props.onSaveSsh({ ssh_key_path: '', ssh_key_passphrase: '' })
    sshForm.value = { ssh_key_path: '', ssh_key_passphrase: '' }
    showPass.value = false
  } catch {
    /* toast handled by parent */
  } finally {
    sshSaving.value = false
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
        <div class="config-item__label">SSH 私钥路径</div>
        <div class="config-item__value">{{ sshPathLabel }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">SSH 私钥口令</div>
        <div class="config-item__value">{{ sshPassLabel }}</div>
      </div>
      <div class="config-item">
        <div class="config-item__label">加密库文件名</div>
        <div class="config-item__value"><code>{{ appConfig.vault_file_name }}</code></div>
      </div>
    </div>

    <!-- 编辑模式 -->
    <div v-else class="edit-stack">
      <form class="config-form" @submit.prevent="submit">
        <h3 class="subsection-title">基本配置</h3>

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
            {{ saving ? '保存中…' : '保存基本配置' }}
          </button>
        </div>
      </form>

      <form class="config-form config-form--ssh" @submit.prevent="saveSSH">
        <h3 class="subsection-title">SSH 凭据（仅 go-git 模式生效）</h3>
        <p class="subsection-desc">
          修复 macOS 上 <code>ssh: handshake failed: EOF</code>：为应用单独指定一把 SSH 私钥；若私钥被口令加密，一并填写口令。
          留空两项表示回退到自动探测（<code>ssh-agent</code> / <code>~/.ssh/id_ed25519</code> / <code>id_ecdsa</code> / <code>id_rsa</code>）。
        </p>

        <div v-if="sshOnlyForGoGit" class="note">
          当前 Git 客户端为 <strong>exec</strong>，这里的 SSH 凭据不会生效；把基本配置切到 <strong>go-git</strong> 后再使用。
        </div>

        <div class="config-form__item">
          <label class="config-form__label" for="cfg-ssh-key">SSH 私钥路径</label>
          <input
            id="cfg-ssh-key"
            v-model="sshForm.ssh_key_path"
            class="input"
            type="text"
            placeholder="/Users/&lt;你&gt;/.ssh/id_pwdmgr（可留空）"
            autocomplete="off"
            spellcheck="false"
          />
          <div class="hint">绝对路径指向一把 SSH 私钥。建议为本应用单独生成一把 <code>ssh-keygen -t ed25519 -f ~/.ssh/id_pwdmgr -N ""</code>。</div>
        </div>

        <div class="config-form__item">
          <label class="config-form__label" for="cfg-ssh-pass">
            SSH 私钥口令
            <span v-if="appConfig.ssh_key_has_pass" class="badge">已保存</span>
          </label>
          <div class="input-row">
            <input
              id="cfg-ssh-pass"
              v-model="sshForm.ssh_key_passphrase"
              class="input input-row__input"
              :type="showPass ? 'text' : 'password'"
              placeholder="留空 = 私钥未加密；非空 = 加密私钥的口令"
              autocomplete="new-password"
              spellcheck="false"
            />
            <button
              type="button"
              class="btn btn--ghost btn--icon"
              :title="showPass ? '隐藏' : '显示'"
              @click="showPass = !showPass"
            >
              <svg v-if="!showPass" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
            </button>
          </div>
          <div class="hint">
            点 <strong>保存 SSH 凭据</strong> 会把当前路径与口令完整覆盖写入；口令框留空表示"私钥未加密"，<strong>不是</strong>"保持旧口令不变"。
            口令以明文存入 <code>pwdmgr.config.json</code>（文件权限 0600）；若有更高安全要求，建议使用未加密的独立私钥，或把 Git 客户端切回 <strong>exec</strong> 让系统 <code>git</code> + Keychain 管理。
          </div>
        </div>

        <div class="config-form__actions">
          <button
            type="button"
            class="btn btn--ghost"
            :disabled="sshSaving || (!appConfig.ssh_key_path && !appConfig.ssh_key_has_pass)"
            @click="clearSSH"
            title="清空后端已保存的 ssh_key_path 与 ssh_key_passphrase"
          >清空已保存</button>
          <button type="submit" class="btn btn--primary" :disabled="sshSaving">
            <svg v-if="sshSaving" class="spinner" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
            {{ sshSaving ? '保存中…' : '保存 SSH 凭据' }}
          </button>
        </div>
      </form>
    </div>

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

.edit-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 18px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, CanvasText 8%, transparent);
  background: color-mix(in srgb, Canvas 92%, transparent);
}

.config-form--ssh {
  border-style: dashed;
}

.subsection-title {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: color-mix(in srgb, CanvasText 85%, transparent);
}

.subsection-desc {
  margin: -6px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: color-mix(in srgb, CanvasText 55%, transparent);
}

.note {
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: #b45309;
  background: color-mix(in srgb, #d97706 10%, transparent);
  border: 1px solid color-mix(in srgb, #d97706 25%, transparent);
}

.config-form__item { display: flex; flex-direction: column; gap: 6px; }

.config-form__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: color-mix(in srgb, CanvasText 70%, transparent);
}

.config-form__label .required { color: #dc2626; margin-left: 2px; }

.badge {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  color: #16a34a;
  background: color-mix(in srgb, #16a34a 14%, transparent);
}

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

.input-row { display: flex; gap: 6px; align-items: stretch; }
.input-row__input { flex: 1; }

.btn--icon {
  padding: 0 10px;
  min-width: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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
