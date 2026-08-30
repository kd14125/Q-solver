<template>
  <div v-if="show" class="modal" id="settings-modal" style="display: flex">
    <div class="modal-content">
      <div class="modal-warning-banner"
        style="background: rgba(255, 169, 64, 0.15); border: 1px solid rgba(255, 169, 64, 0.3); border-radius: 50px; padding: 6px 20px; color: #ffc069; font-size: 12px; display: flex; align-items: center; justify-content: center; margin: 12px auto 4px auto; width: fit-content;">
        ⚠️ 当前窗口已获取焦点，关闭设置后将自动恢复防抢焦模式
      </div>
      <div class="modal-header">
        <div class="tabs">
          <div class="tab" :class="{ active: currentTab === 'general' }" @click="currentTab = 'general'">
            常规设置</div>
          <div class="tab" :class="{ active: currentTab === 'model' }" @click="currentTab = 'model'">截图答题模型
          </div>
          <div class="tab" :class="{ active: currentTab === 'realtime' }" @click="currentTab = 'realtime'">语音面试模型</div>
          <div class="tab" :class="{ active: currentTab === 'params' }" @click="currentTab = 'params'">截图生成参数</div>
          <div class="tab" :class="{ active: currentTab === 'screenshot' }" @click="currentTab = 'screenshot'">截图设置</div>
          <div class="tab" :class="{ active: currentTab === 'resume' }" @click="currentTab = 'resume'">
            简历设置</div>
          <div class="tab" :class="{ active: currentTab === 'account' }" @click="currentTab = 'account'">
            截图提供商</div>
        </div>
        <span class="close-btn" @click="$emit('close')">&times;</span>
      </div>
      <div class="modal-body">
        <div v-show="currentTab === 'account'">
          <ProviderSelect v-model:provider="tempSettings.provider" v-model:apiKey="tempSettings.apiKey"
            v-model:baseURL="tempSettings.baseURL" />
        </div>

        <div v-show="currentTab === 'model'">
          <div class="form-group">
            <div class="model-header">
              <label>模型选择</label>
              <div class="model-actions">
                <button class="btn-icon" @click="$emit('refresh-models')"
                  :disabled="isLoadingModels || !tempSettings.apiKey" title="刷新模型列表">
                  <svg class="action-icon" :class="{ spin: isLoadingModels }" viewBox="0 0 16 16" fill="none">
                    <path d="M14 8a6 6 0 01-10.24 4.24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    <path d="M2 8a6 6 0 0110.24-4.24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    <path d="M14 3v5h-5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                    <path d="M2 13V8h5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
                <button class="btn-icon" @click="$emit('test-connection')"
                  :disabled="isTestingConnection || !tempSettings.model" title="测试模型连通性">
                  <svg v-if="isTestingConnection" class="action-icon spin" viewBox="0 0 16 16" fill="none">
                    <circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="1.5" stroke-dasharray="28 10" stroke-linecap="round"/>
                  </svg>
                  <svg v-else class="action-icon" viewBox="0 0 16 16" fill="none">
                    <path d="M4 3l9 5-9 5V3z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
                  </svg>
                </button>
              </div>
            </div>
            <ModelSelect v-model="tempSettings.model" :models="availableModels" :loading="isLoadingModels" />

            <!-- 连通性测试结果 -->
            <div v-if="connectionStatus" class="connection-status" :class="connectionStatus.type">
              <span class="status-icon">{{ connectionStatus.icon }}</span>
              <span class="status-text">{{ connectionStatus.message }}</span>
            </div>

            <p v-if="!tempSettings.apiKey" class="hint-text warning-hint">
              ⚠️ 请先填写 API Key
            </p>
          </div>

          <!-- 辅助模型 -->
          <div class="form-group">
            <div class="model-header">
              <label>辅助模型</label>
            </div>
            <ModelSelect v-model="tempSettings.assistantModel" :models="availableModels" :loading="isLoadingModels" 
              placeholder="选择辅助模型（可选）" />
            <p class="hint-text">
              💡 用于总结对话生成问题导图，留空则不生成导图
            </p>
          </div>

          <div class="form-group">
            <div class="prompt-header">
              <label for="prompt-text" style="margin-bottom: 0">系统提示词 (Prompt)</label>
              <div class="prompt-tabs">
                <div class="prompt-tab" :class="{ active: promptTab === 'edit' }" @click="promptTab = 'edit'">编辑
                </div>
                <div class="prompt-tab" :class="{ active: promptTab === 'preview' }" @click="promptTab = 'preview'">预览
                </div>
              </div>
            </div>

            <textarea v-show="promptTab === 'edit'" id="prompt-text" class="prompt-textarea" rows="10"
              v-model="tempSettings.prompt" placeholder="请输入提示词 (支持 Markdown)..."></textarea>

            <div v-show="promptTab === 'preview'" class="prompt-preview markdown-body" v-html="renderedPrompt">
            </div>
          </div>
        </div>

        <div v-show="currentTab === 'params'">
          <LLMParamsConfig v-model:temperature="tempSettings.temperature" v-model:topP="tempSettings.topP"
            v-model:topK="tempSettings.topK" v-model:maxTokens="tempSettings.maxTokens"
            v-model:thinkingBudget="tempSettings.thinkingBudget" />
        </div>

        <div v-show="currentTab === 'realtime'">
          <div class="form-group">
            <div class="setting-row">
              <div class="setting-info"><span class="setting-title">启用语音面试模式</span><span class="setting-desc">采集系统声音并实时显示面试问题和文字答案</span></div>
              <label class="switch"><input type="checkbox" v-model="tempSettings.useLiveApi"><span class="slider round"></span></label>
            </div>
          </div>

          <div class="form-group context-setting">
            <div class="setting-row">
              <div class="setting-info"><span class="setting-title">Qwen3.5 Omni Realtime</span><span class="setting-desc">semantic VAD 自动判断问题结束，仅输出文字</span></div>
              <label class="switch"><input type="checkbox" v-model="tempSettings.realtimeEnabled" @change="onRealtimeToggle"><span class="slider round"></span></label>
            </div>
          </div>

          <template v-if="tempSettings.realtimeEnabled">
            <div class="form-group"><label>Realtime API Key</label><input class="form-input" type="password" v-model="tempSettings.realtimeAPIKey" autocomplete="off" /></div>
            <div class="form-group"><label>Workspace ID</label><input class="form-input" v-model.trim="tempSettings.realtimeWorkspaceID" placeholder="阿里云 Model Studio Workspace ID" /></div>
            <div class="form-group"><label>地域</label><input class="form-input" v-model.trim="tempSettings.realtimeRegion" placeholder="cn-beijing" /></div>
            <div class="form-group"><label>Realtime Base URL（可选）</label><input class="form-input" v-model.trim="tempSettings.realtimeBaseURL" placeholder="留空时根据 Workspace ID 和地域自动生成" /></div>
            <div class="form-group"><label>模型</label><input class="form-input" v-model.trim="tempSettings.realtimeModel" placeholder="qwen3.5-omni-plus-realtime" /></div>
            <div class="form-group"><label>语音面试系统提示词</label><textarea class="prompt-textarea" rows="10" v-model="tempSettings.realtimePrompt"></textarea><p class="hint-text">默认生成候选人可直接说出口的短回答：普通问题约 30–60 秒，复杂技术题通常不超过 90 秒。</p></div>
            <div class="form-group realtime-grid">
              <label>Temperature<input class="form-input" type="number" min="0" max="1.99" step="0.1" v-model.number="tempSettings.realtimeTemperature" /></label>
              <label>Top P<input class="form-input" type="number" min="0.01" max="1" step="0.05" v-model.number="tempSettings.realtimeTopP" /></label>
              <label>Top K<input class="form-input" type="number" min="0" step="1" v-model.number="tempSettings.realtimeTopK" /></label>
              <label>Max Tokens<input class="form-input" type="number" min="1" step="100" v-model.number="tempSettings.realtimeMaxTokens" /></label>
            </div>
            <p class="hint-text">推荐值：Temperature 0.4、Top P 0.8、Top K 20、Max Tokens 600，兼顾自然口语、稳定性和回答长度。</p>
            <div class="form-group"><label>VAD 类型</label><select class="form-input" v-model="tempSettings.realtimeVADType"><option value="semantic_vad">semantic_vad</option><option value="server_vad">server_vad</option></select></div>
            <div class="form-group realtime-grid">
              <label>VAD 阈值<input class="form-input" type="number" min="-1" max="1" step="0.1" v-model.number="tempSettings.realtimeVADThreshold" /></label>
              <label>静音判定（ms）<input class="form-input" type="number" min="200" max="6000" step="100" v-model.number="tempSettings.realtimeSilenceDurationMs" /></label>
            </div>
            <button class="btn-primary" type="button" @click="$emit('test-realtime-connection')" :disabled="isTestingRealtime || !tempSettings.realtimeAPIKey || (!tempSettings.realtimeWorkspaceID && !tempSettings.realtimeBaseURL)">{{ isTestingRealtime ? '连接中…' : '测试语音 API 连接' }}</button>
            <div v-if="realtimeConnectionStatus" class="connection-status" :class="realtimeConnectionStatus.type"><span class="status-icon">{{ realtimeConnectionStatus.icon }}</span><span class="status-text">{{ realtimeConnectionStatus.message }}</span></div>
            <RAGSettings :settings="tempSettings" />
          </template>

          <div class="form-group context-setting" style="margin-top: 18px;">
            <div class="setting-row">
              <div class="setting-info"><span class="setting-title">兼容第三方 STT（每 5 秒）</span><span class="setting-desc">上传固定音频段，再使用截图答题文本模型回答；不能与 Qwen Realtime 同时启用</span></div>
              <label class="switch"><input type="checkbox" v-model="tempSettings.sttEnabled" @change="onSTTToggle"><span class="slider round"></span></label>
            </div>
          </div>
          <template v-if="tempSettings.sttEnabled">
            <div class="form-group"><label>语音转文字 API 地址</label><input class="form-input" v-model.trim="tempSettings.sttBaseURL" placeholder="例如 https://api.openai.com/v1" /></div>
            <div class="form-group"><label>语音转文字 API Key</label><input class="form-input" type="password" v-model="tempSettings.sttAPIKey" autocomplete="off" /></div>
            <div class="form-group">
              <div class="model-header"><label>语音识别模型</label><button class="btn-icon" @click="$emit('refresh-stt-models')" :disabled="isLoadingSTTModels || !tempSettings.sttAPIKey || !tempSettings.sttBaseURL" title="刷新语音识别模型列表">↻</button></div>
              <ModelSelect v-if="sttAvailableModels.length" v-model="tempSettings.sttModel" :models="sttAvailableModels" :loading="isLoadingSTTModels" placeholder="选择语音识别模型" />
              <input class="form-input" v-model.trim="tempSettings.sttModel" placeholder="whisper-1" />
            </div>
            <div class="form-group"><label>识别语言</label><input class="form-input" v-model.trim="tempSettings.sttLanguage" placeholder="zh" /></div>
            <div class="setting-row"><div class="setting-info"><span class="setting-title">朗读回答</span><span class="setting-desc">仅兼容 STT 模式使用系统语音朗读</span></div><label class="switch"><input type="checkbox" v-model="tempSettings.voiceReply"><span class="slider round"></span></label></div>
          </template>
        </div>

        <div v-show="currentTab === 'general'">
          <div class="form-group">
            <div class="context-setting">
              <div class="setting-row">
                <div class="setting-info">
                  <span class="setting-title">保存上下文</span>
                  <span class="setting-desc">开启后，每次对话将包含之前的历史记录</span>
                </div>
                <label class="switch">
                  <input type="checkbox" v-model="tempSettings.keepContext">
                  <span class="slider round"></span>
                </label>
              </div>

            </div>
          </div>

          <div class="form-group">
            <label>快捷键配置 {{ isMacOS ? '(macOS 不支持自定义)' : '(点击录制)' }}</label>
            <div class="shortcut-list">
              <div class="shortcut-item" v-for="key in shortcutActions" :key="key.action">
                <span>{{ key.label }}</span>
                <button class="btn-record" :class="{ recording: recordingAction === key.action, disabled: isMacOS }"
                  @click="!isMacOS && $emit('record-key', key.action)"
                  :title="isMacOS ? 'macOS 使用预设快捷键，不支持自定义' : '点击录制新快捷键'">
                  {{ recordingAction === key.action ? recordingText : (tempShortcuts[key.action]?.keyName ||
                    (isMacOS ? key.macDefault : key.default)) }}
                </button>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label>界面主题</label>
            <div class="theme-options" role="radiogroup" aria-label="界面主题">
              <button type="button" class="theme-option" :class="{ active: tempSettings.theme === 'dark' }"
                role="radio" :aria-checked="tempSettings.theme === 'dark'" @click="tempSettings.theme = 'dark'">
                <span class="theme-preview dark-preview" aria-hidden="true">
                  <span></span><span></span><span></span>
                </span>
                <span class="theme-option-copy">
                  <strong>🌙 夜间模式</strong>
                  <small>深色低亮度，适合夜间使用</small>
                </span>
              </button>
              <button type="button" class="theme-option" :class="{ active: tempSettings.theme === 'light' }"
                role="radio" :aria-checked="tempSettings.theme === 'light'" @click="tempSettings.theme = 'light'">
                <span class="theme-preview light-preview" aria-hidden="true">
                  <span></span><span></span><span></span>
                </span>
                <span class="theme-option-copy">
                  <strong>☀️ 白天模式</strong>
                  <small>浅色高对比，适合明亮环境</small>
                </span>
              </button>
            </div>
            <p class="hint-text">点击即可即时预览，保存后下次启动仍会使用所选主题。</p>
          </div>

          <div class="form-group">
            <label for="opacity-slider">窗口透明度: <span>{{ Math.round(tempSettings.transparency * 100) }}%</span></label>
            <input type="range" id="opacity-slider" min="0.0" max="1.0" step="0.05"
              v-model.number="tempSettings.transparency" />
          </div>

          <div class="form-group">
            <label for="ai-font-size">AI 回复字体大小: <span>{{ tempSettings.aiFontSize }}px</span></label>
            <input type="range" id="ai-font-size" min="10" max="32" step="1"
              v-model.number="tempSettings.aiFontSize" />
            <p class="hint-text">只调整 AI 回复正文和代码块字体，不影响设置界面。</p>
          </div>

          <div class="form-group">
            <label>AI 回复字体颜色</label>
            <div class="theme-options" role="radiogroup" aria-label="AI 回复字体颜色">
              <button type="button" class="theme-option" :class="{ active: tempSettings.aiTextColor !== 'black' }"
                role="radio" :aria-checked="tempSettings.aiTextColor !== 'black'" @click="tempSettings.aiTextColor = 'white'">
                <span class="font-color-preview white-font-preview" aria-hidden="true">Aa</span>
                <span class="theme-option-copy">
                  <strong>白色字体</strong>
                  <small>适合深色或透明背景</small>
                </span>
              </button>
              <button type="button" class="theme-option" :class="{ active: tempSettings.aiTextColor === 'black' }"
                role="radio" :aria-checked="tempSettings.aiTextColor === 'black'" @click="tempSettings.aiTextColor = 'black'">
                <span class="font-color-preview black-font-preview" aria-hidden="true">Aa</span>
                <span class="theme-option-copy">
                  <strong>黑色字体</strong>
                  <small>适合浅色或白色背景</small>
                </span>
              </button>
            </div>
            <p class="hint-text">选择后立即预览，正文、标题、列表和代码块会统一切换。</p>
          </div>

          <div class="form-group">
            <label for="ai-text-opacity">AI 回复文字透明度: <span>{{ Math.round(tempSettings.aiTextTransparency * 100) }}%</span></label>
            <input type="range" id="ai-text-opacity" min="0" max="0.95" step="0.05"
              v-model.number="tempSettings.aiTextTransparency" />
            <p class="hint-text">只调整回答正文、标题和代码文字的透明度，不影响窗口背景。</p>
          </div>

          <div class="form-group">
            <div class="setting-row">
              <div class="setting-info">
                <span class="setting-title">隐藏顶部工具栏</span>
                <span class="setting-desc">隐藏快捷键提示、设置、状态和退出按钮，需要使用快捷键操作。</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="tempSettings.hideTopBar">
                <span class="slider round"></span>
              </label>
            </div>
          </div>

          <div class="form-group">
            <div class="setting-row">
              <div class="setting-info">
                <span class="setting-title">隐藏历史问题栏</span>
                <span class="setting-desc">隐藏左侧历史问题列表，让回答区域占满窗口。</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="tempSettings.hideHistoryPanel">
                <span class="slider round"></span>
              </label>
            </div>
          </div>

          <div class="form-group">
            <div class="setting-row">
              <div class="setting-info">
                <span class="setting-title">代码块自动换行</span>
                <span class="setting-desc">开启后代码过长时自动折行，不需要左右滚动</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="tempSettings.codeWrap">
                <span class="slider round"></span>
              </label>
            </div>
          </div>
        </div>

        <div v-show="currentTab === 'screenshot'">
          <ScreenshotSettings :modelValue="tempSettings" @update:modelValue="Object.assign(tempSettings, $event)" />
        </div>

        <div v-show="currentTab === 'resume'" style="height: 100%">
          <ResumeImport :resumePath="tempSettings.resumePath" :rawContent="resumeRawContent"
            :isParsing="isResumeParsing" :currentModel="tempSettings.model"
            v-model:useMarkdownResume="tempSettings.useMarkdownResume"
            @update:rawContent="$emit('update:resumeRawContent', $event)" @select-resume="$emit('select-resume')"
            @clear-resume="$emit('clear-resume')" @parse-resume="$emit('parse-resume')" />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-primary" @click="$emit('save')">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import ResumeImport from './ResumeImport.vue'
import ScreenshotSettings from './ScreenshotSettings.vue'
import ProviderSelect from './ProviderSelect.vue'
import ModelSelect from './ModelSelect.vue'
import LLMParamsConfig from './LLMParamsConfig.vue'
import RAGSettings from './RAGSettings.vue'

const props = defineProps({
  show: Boolean,
  tempSettings: Object,
  tempShortcuts: Object,
  shortcutActions: Array,
  recordingAction: String,
  recordingText: String,
  availableModels: Array,
  isLoadingModels: Boolean,
  sttAvailableModels: {
    type: Array,
    default: () => []
  },
  isLoadingSTTModels: Boolean,
  isTestingConnection: Boolean,
  connectionStatus: Object,
  isTestingRealtime: Boolean,
  realtimeConnectionStatus: Object,
  renderedPrompt: String,
  resumeRawContent: String,
  isResumeParsing: Boolean,
  isMacOS: Boolean,
  activeTab: {
    type: String,
    defaut: 'general'
  }
})

const emit = defineEmits([
  'close',
  'save',
  'refresh-models',
  'refresh-stt-models',
  'test-connection',
  'test-realtime-connection',
  'record-key',
  'select-resume',
  'clear-resume',
  'parse-resume',
  'update:resumeRawContent',
  'update:activeTab'
])

const currentTab = computed({
  get: () => props.activeTab || 'general',
  set: (val) => emit('update:activeTab', val)
})

const promptTab = ref('edit')

function onRealtimeToggle() {
  if (props.tempSettings.realtimeEnabled) {
    props.tempSettings.sttEnabled = false
    props.tempSettings.voiceReply = false
  }
}

function onSTTToggle() {
  if (props.tempSettings.sttEnabled) {
    props.tempSettings.realtimeEnabled = false
  }
}
</script>

<style scoped>
.realtime-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.realtime-grid label {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.theme-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 8px;
}

.theme-option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--transition-fast), background var(--transition-fast), box-shadow var(--transition-fast);
}

.theme-option:hover {
  border-color: var(--border-hover);
  background: var(--bg-card-hover);
}

.theme-option.active {
  border-color: var(--color-primary);
  background: var(--color-primary-light);
  box-shadow: var(--shadow-focus);
}

.theme-preview {
  flex: 0 0 48px;
  height: 36px;
  display: flex;
  align-items: flex-end;
  gap: 3px;
  padding: 6px;
  border-radius: 7px;
  box-sizing: border-box;
  box-shadow: inset 0 0 0 1px rgba(100, 116, 139, 0.18);
}

.theme-preview span {
  display: block;
  height: 5px;
  border-radius: 999px;
}

.theme-preview span:nth-child(1) { width: 12px; }
.theme-preview span:nth-child(2) { width: 8px; }
.theme-preview span:nth-child(3) { width: 14px; }
.dark-preview { background: #111827; }
.dark-preview span { background: #94a3b8; }
.light-preview { background: #f8fafc; }
.light-preview span { background: #475569; }

.font-color-preview {
  flex: 0 0 48px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 7px;
  background: linear-gradient(135deg, #64748b, #94a3b8);
  box-shadow: inset 0 0 0 1px rgba(100, 116, 139, 0.18);
  font-size: 16px;
  font-weight: 700;
}

.white-font-preview { color: #ffffff; }
.black-font-preview { color: #000000; }

.theme-option-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.theme-option-copy strong {
  font-size: 13px;
  font-weight: 600;
}

.theme-option-copy small {
  color: var(--text-tertiary);
  font-size: 10px;
  line-height: 1.35;
}

@media (max-width: 520px) {
  .theme-options {
    grid-template-columns: 1fr;
  }
}
</style>
