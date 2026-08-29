<template>
  <TopBar :shortcuts="shortcuts" :activeButtons="activeButtons" :isClickThrough="isClickThrough"
    :statusIcon="statusIcon" :statusText="statusText" :settings="settings" :isStealthMode="isStealthMode"
    :isMacOS="isMacOS"
    @openSettings="openSettings" @quit="quit" />

  <!-- 发布版兼容：手动截图缓存面板（最多 3 张） -->
  <div v-if="!settings.useLiveApi && pendingScreenshots.length > 0" class="screenshot-dock">
    <div class="screenshot-dock-head">
      <span>截图 {{ pendingScreenshots.length }}/3</span>
      <button v-if="pendingScreenshots.length" class="screenshot-clear" @click="clearPendingScreenshots">清空</button>
    </div>
    <div v-if="pendingScreenshots.length" class="screenshot-thumbs">
      <div v-for="(shot, idx) in pendingScreenshots" :key="idx" class="screenshot-thumb">
        <img :src="shot" :alt="`截图 ${idx + 1}`" />
        <button @click="removePendingScreenshot(idx)" :aria-label="`删除截图 ${idx + 1}`">×</button>
      </div>
    </div>
    <div class="screenshot-actions">
      <button @click="takeScreenshot" :disabled="pendingScreenshots.length >= 3 || isLoading">截图</button>
      <button class="primary" @click="sendScreenshots" :disabled="isLoading">发送</button>
    </div>
  </div>

  <!-- Live API 模式优先 -->
  <LiveView v-if="settings.useLiveApi" />

  <WelcomeView v-else-if="!hasStarted || history.length === 0" :solveShortcut="solveShortcut"
    :toggleShortcut="shortcuts.toggle?.keyName || 'Alt+H'" :initStatus="initStatus" />

  <div v-else id="main-interface" class="main-interface" :class="{ visible: mainVisible }">
    <div class="left-panel" id="history-list">
      <div v-if="history.length === 0" class="history-empty">
        <span class="empty-icon">📝</span>
        <span class="empty-text">暂无记录</span>
      </div>
      <HistoryItem v-for="(h, idx) in history" :key="idx" :summary="getSummary(h)" :time="h.time"
        :isActive="idx === activeHistoryIndex" :isFirst="idx === 0" :previewHtml="renderMarkdown(getSummary(h))"
        :roundsCount="getRoundsCount(h)" @select="selectHistory(idx)" @delete="deleteHistory(idx)"
        @export-image="exportImage(idx)" />
    </div>
    <div class="right-panel">
      <EmptyState v-if="history.length === 0 && !isLoading && !errorState.show" :shortcut="solveShortcut" />
      <ErrorView v-else-if="errorState.show" :errorState="errorState" :solveShortcut="solveShortcut" />
      <LoadingView v-else-if="isLoading" />
      <div v-else id="content" class="markdown-body">
        <template v-for="(round, idx) in currentRounds" :key="idx">
          <div class="chat-round">
            <!-- 思维链区域（Cherry Studio 风格） -->
            <div v-if="round.thinking" class="thinking-block" :class="{ expanded: round.thinkingExpanded }">
              <div class="thinking-header" @click="round.thinkingExpanded = !round.thinkingExpanded">
                <div class="thinking-left">
                  <span class="thinking-icon">💭</span>
                  <span class="thinking-title">思考中</span>
                  <span class="thinking-duration" v-if="round.thinkingDuration">
                    {{ formatDuration(round.thinkingDuration) }}
                  </span>
                </div>
                <span class="thinking-toggle">{{ round.thinkingExpanded ? '收起' : '展开' }}</span>
              </div>
              <div class="thinking-preview" v-if="!round.thinkingExpanded">
                <div class="thinking-preview-text">{{ getThinkingPreview(round.thinking) }}</div>
              </div>
              <div class="thinking-content" v-else v-html="renderMarkdown(round.thinking)"></div>
            </div>
            <!-- 正文回复 -->
            <div class="ai-response" v-html="renderMarkdown(round.aiResponse)"></div>
          </div>
          <hr v-if="idx < currentRounds.length - 1" class="round-divider" />
        </template>
        <!-- 思维中状态 -->
        <div v-if="isThinking" class="thinking-loading">
          <div class="thinking-indicator">
            <span class="pulse-dot"></span>
            <span class="text">正在思考中...</span>
            <span class="thinking-timer">{{ formatDuration(thinkingTimer) }}</span>
          </div>
        </div>
        <!-- 追加加载状态 -->
        <div v-if="isAppending && !isThinking" class="append-loading">
          <div class="ai-icon">
            <div class="ai-icon-inner"></div>
          </div>
          <span class="text">AI 正在回复</span>
          <div class="wave-dots">
            <span></span><span></span><span></span>
          </div>
        </div>
      </div>
    </div>
  </div>


  <!-- Settings Modal -->
  <SettingsModal :show="uiState.showSettings" :tempSettings="tempSettings" :tempShortcuts="tempShortcuts"
    :shortcutActions="shortcutActions" :recordingAction="recordingAction" :recordingText="recordingText"
    :availableModels="uiState.availableModels" :isLoadingModels="uiState.isLoadingModels"
    :sttAvailableModels="uiState.sttAvailableModels" :isLoadingSTTModels="uiState.isLoadingSTTModels"
    :isTestingConnection="uiState.isTestingConnection" :connectionStatus="uiState.connectionStatus"
    :isTestingRealtime="uiState.isTestingRealtime" :realtimeConnectionStatus="uiState.realtimeConnectionStatus"
    :renderedPrompt="renderedPrompt" :resumeRawContent="resumeState.rawContent" :isResumeParsing="resumeState.isParsing"
    :isMacOS="isMacOS"
    v-model:activeTab="uiState.activeTab"
    @close="closeSettings" @save="saveSettings" @refresh-models="refreshModels" @refresh-stt-models="refreshSTTModels" @test-connection="testConnection" @test-realtime-connection="testRealtimeConnection"
    @record-key="recordKey" @select-resume="selectResume" @clear-resume="clearResume" @parse-resume="parseResume"
    @update:resumeRawContent="val => resumeState.rawContent = val" />

  <!-- 简历兼容性确认弹窗 -->
  <div v-if="showResumeWarning" class="modal" style="display: flex">
    <div class="resume-warning-dialog">
      <div class="warning-icon">⚠️</div>
      <div class="warning-title">简历可能无法发送</div>
      <div class="warning-message">
        当前模型不支持 PDF，且简历未解析为 Markdown。<br />
        继续解题将跳过简历内容。
      </div>
      <div class="warning-actions">
        <button class="btn-secondary" @click="cancelSolve">取消</button>
        <button class="btn-primary" @click="continueSolve">继续解题</button>
      </div>
    </div>
  </div>

  <div id="toast-container">
    <div v-for="(t, i) in toasts" :key="t.id || i" class="toast" :class="[t.type, { show: t.show }]">{{ t.text }}
    </div>
  </div>

  <!-- 窗口调整手柄 -->
  <ResizeHandle />

</template>

<script setup>
import { reactive, ref, onMounted, watch, nextTick, computed } from 'vue'
import SettingsModal from './components/SettingsModal.vue'
import WelcomeView from './components/WelcomeView.vue'
import ErrorView from './components/ErrorView.vue'
import LoadingView from './components/LoadingView.vue'
import TopBar from './components/TopBar.vue'
import HistoryItem from './components/HistoryItem.vue'
import EmptyState from './components/EmptyState.vue'
import LiveView from './components/LiveView.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import { EventsOn, Quit } from '../wailsjs/runtime/runtime'
import { StopRecordingKey, SelectResume, ClearResume, RestoreFocus, RemoveFocus, ParseResume, GetInitStatus, TriggerScreenshot, TriggerSend, RemoveScreenshot, ClearScreenshots } from '../wailsjs/go/main/App'

import { useUI } from './composables/useUI'
import { useStatus } from './composables/useStatus'
import { useShortcuts } from './composables/useShortcuts'
import { useSettings } from './composables/useSettings'
import { useSolution } from './composables/useSolution'
import { supportsVision, supportsPDF } from './utils/modelCapabilities'

// 样式导入
import './App.global.css'
import './App.scoped.css'

const uiState = reactive({
  showSettings: false,
  activeTab: 'general',
  availableModels: [],
  isLoadingModels: false,
  sttAvailableModels: [],
  isLoadingSTTModels: false,
  isModelDropdownOpen: false,
  promptTab: 'edit',
  isTestingConnection: false,
  connectionStatus: null,
  isTestingRealtime: false,
  realtimeConnectionStatus: null,
})

const {
  toasts, activeButtons, isClickThrough, mainVisible, isStealthMode, hasStarted,
  showToast, flash, quit
} = useUI()

const {
  shortcuts, tempShortcuts, recordingAction, recordingText, shortcutActions, recordKey, isMacOS
} = useShortcuts()

// Settings callbacks placeholder
const settingsCallbacks = {}

const {
  settings, tempSettings, renderedPrompt, maskedKey,
  loadSettings, refreshModels, refreshSTTModels, testConnection, testRealtimeConnection, fetchModels, saveSettings, resetTempSettings, openSettings: initSettings
} = useSettings(shortcuts, tempShortcuts, uiState, settingsCallbacks)

watch(() => settings.aiFontSize, (size) => {
  const value = Math.min(32, Math.max(10, Number(size) || 14))
  document.documentElement.style.setProperty('--ai-font-size', `${value}px`)
}, { immediate: true })

watch(() => settings.codeWrap, (enabled) => {
  document.documentElement.classList.toggle('code-wrap-enabled', enabled === true)
}, { immediate: true })

const resumeState = reactive({
  rawContent: '',
  isParsing: false
})

const pendingScreenshots = ref([])
async function takeScreenshot() { await TriggerScreenshot() }
async function sendScreenshots() { await TriggerSend() }
async function removePendingScreenshot(index) { await RemoveScreenshot(index) }
async function clearPendingScreenshots() { await ClearScreenshots() }

watch(() => resumeState.rawContent, (newVal) => {
  tempSettings.resumeContent = newVal || ''
})

// 简历兼容性警告弹窗
const showResumeWarning = ref(false)
let pendingSolveCallback = null

function cancelSolve() {
  showResumeWarning.value = false
  pendingSolveCallback = null
}

function continueSolve() {
  showResumeWarning.value = false
  if (pendingSolveCallback) {
    pendingSolveCallback()
    pendingSolveCallback = null
  }
}

async function selectResume() {
  const path = await SelectResume()
  if (path) {
    tempSettings.resumePath = path
    resumeState.rawContent = '' // Reset parsed content on new file
    showToast('简历已选择', 'success')
  }
}

async function clearResume() {
  await ClearResume()
  tempSettings.resumePath = ''
  resumeState.rawContent = ''
}
async function parseResume() {
  if (!tempSettings.resumePath) return

  resumeState.isParsing = true
  try {
    const result = await ParseResume()
    resumeState.rawContent = result
    showToast('简历解析成功', 'success')
  } catch (e) {
    console.error(e)
    showToast('解析失败: ' + e, 'error')
  } finally {
    resumeState.isParsing = false
  }
}

const {
  statusText, statusIcon, resetStatus
} = useStatus(settings)



const {
  currentRounds, history, activeHistoryIndex, isLoading, isAppending, isThinking, shouldOverwriteHistory,
  errorState, renderMarkdown, getFullContent, getSummary, getRoundsCount, selectHistory, handleStreamStart, handleStreamChunk, handleThinkingChunk, handleSolution, setStreamBuffer,
  setUserScreenshot, deleteHistory, exportImage
} = useSolution(settings)

// 思考时间计时器
const thinkingTimer = ref(0)
let thinkingTimerInterval = null

watch(isThinking, (val) => {
  if (val) {
    thinkingTimer.value = 0
    thinkingTimerInterval = setInterval(() => {
      thinkingTimer.value += 0.1
    }, 100)
  } else {
    if (thinkingTimerInterval) {
      clearInterval(thinkingTimerInterval)
      thinkingTimerInterval = null
    }
  }
})

// 格式化时长
function formatDuration(seconds) {
  if (!seconds || seconds < 0) return ''
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`
  }
  const mins = Math.floor(seconds / 60)
  const secs = (seconds % 60).toFixed(0)
  return `${mins}m ${secs}s`
}

// 获取思考预览（最后两行，实时滚动）
function getThinkingPreview(thinking) {
  if (!thinking) return ''
  const lines = thinking.split('\n').filter(l => l.trim())
  // 取最后两行
  const lastLines = lines.slice(-2)
  const preview = lastLines.join(' ')
  if (preview.length > 120) {
    return '...' + preview.substring(preview.length - 120)
  }
  return lines.length > 2 ? '...' + preview : preview
}

// Populate callbacks

settingsCallbacks.resetStatus = resetStatus
settingsCallbacks.showToast = showToast

settingsCallbacks.closeSettings = closeSettings

function openSettings() {
  RestoreFocus()
  // 初始化临时设置
  initSettings()


  // 加载模型列表
  if (settings.apiKey) {
    fetchModels(settings.apiKey, settings.baseURL)
  }
  if (tempSettings.sttEnabled && tempSettings.sttAPIKey && tempSettings.sttBaseURL) {
    refreshSTTModels()
  }

  // 加载简历内容
  if (settings.resumeContent) {
    resumeState.rawContent = settings.resumeContent
  }

  uiState.showSettings = true
}

function closeSettings() {
  RemoveFocus()
  uiState.showSettings = false
  if (recordingAction.value) {
    StopRecordingKey()
  }
  recordingAction.value = null
  recordingText.value = ''
  // 恢复所有临时设置到原值（包括透明度）
  resetTempSettings()
}

const solveShortcut = computed(() => shortcuts.send?.keyName || shortcuts.solve?.keyName || 'F7')

const initStatus = ref('initializing')
// Lifecycle
onMounted(() => {
  // localStorage.clear()
  GetInitStatus().then(status => {
    initStatus.value = status
  })

  EventsOn('init-status', (status) => {
    initStatus.value = status
  })

  loadSettings().then(() => {
    resetStatus()
  })

  // Event Listeners
  EventsOn('key-recorded', (data) => {
    if (data && data.action) {
      if (tempShortcuts[data.action]) {
        tempShortcuts[data.action].keyName = data.keyName
        tempShortcuts[data.action].vkCode = data.comboID
      } else {
        tempShortcuts[data.action] = { keyName: data.keyName, vkCode: data.comboID }
      }

      if (recordingAction.value === data.action) {
        recordingText.value = data.keyName
      }
    }
  })

  EventsOn('shortcut-error', async (msg) => {
    showToast(msg, 'error', 2000)
    const targetAction = recordingAction.value
    recordingAction.value = null
    recordingText.value = ''
    StopRecordingKey()
    if (!targetAction) return

    try {
      if (shortcuts[targetAction] && shortcuts[targetAction].keyName) {
        tempShortcuts[targetAction] = JSON.parse(JSON.stringify(shortcuts[targetAction]))
      } else {
        delete tempShortcuts[targetAction]
      }
    } catch (e) {
      console.error("回滚配置失败", e)
    }
  })

  EventsOn('shortcut-saved', (action) => {
    if (recordingAction.value === action) {
      recordingAction.value = null
      showToast('快捷键已保存', 'success')
    }
  })

  // 接收用户截图用于导出功能
  EventsOn('user-message', (screenshot) => {
    setUserScreenshot(screenshot)
  })

  EventsOn('screenshot-taken', (screenshot) => {
    if (screenshot && pendingScreenshots.value.length < 3) pendingScreenshots.value.push(screenshot)
  })
  EventsOn('screenshot-removed', (index) => {
    if (Number.isInteger(index)) pendingScreenshots.value.splice(index, 1)
  })
  EventsOn('screenshots-cleared', () => {
    pendingScreenshots.value = []
  })

  EventsOn('start-solving', () => {
    // 检查简历兼容性
    const hasPdfResume = settings.resumePath && !settings.useMarkdownResume
    const hasMarkdownContent = settings.resumeContent && settings.useMarkdownResume
    const modelCanHandle = supportsVision(settings.model) || supportsPDF(settings.model)

    if (hasPdfResume && !hasMarkdownContent && !modelCanHandle) {
      // 模型不支持，弹窗确认
      pendingSolveCallback = proceedWithSolve
      showResumeWarning.value = true
      return
    }

    proceedWithSolve()
  })

  function proceedWithSolve() {
    errorState.show = false
    flash('solve')
    statusText.value = '正在思考...'
    statusIcon.value = '🟡'
    mainVisible.value = true
    hasStarted.value = true

    if (settings.keepContext && history.value.length > 0 && activeHistoryIndex.value === 0) {
      isLoading.value = false
      isAppending.value = true
      // 使用 setTimeout 确保 DOM 更新后滚动
      setTimeout(() => {
        const contentDiv = document.getElementById('content')
        if (contentDiv) {
          contentDiv.scrollTop = contentDiv.scrollHeight
        }
      }, 50)
    } else {
      isLoading.value = true
      isAppending.value = false
    }
  }

  EventsOn('toggle-visibility', (isVisibleToCapture) => {
    flash('toggle')
    isStealthMode.value = isVisibleToCapture
    if (isVisibleToCapture) {
      showToast('隐身模式已开启 (录屏不可见)', 'info')
    } else {
      showToast('隐身模式已关闭 (录屏可见)', 'success')
    }
  })

  EventsOn('solution', (data) => {
    statusText.value = '解题完成'
    statusIcon.value = '📝'
    handleSolution(data)

  })

  EventsOn('copy-code', () => {
    const old = statusText.value
    statusText.value = '已复制'
    setTimeout(() => (statusText.value = old), 2000)
  })

  EventsOn('click-through-state', (enabled) => {
    isClickThrough.value = enabled
    const el = document.getElementById('main-interface')
    if (el) el.style.pointerEvents = enabled ? "none" : "auto"
  })

  EventsOn("scroll-content", (direction) => {
    const contentDiv = document.getElementById('content')
    if (!contentDiv) return
    const scrollAmount = 50;
    if (direction === "up") {
      contentDiv.scrollBy({ top: -scrollAmount, behavior: 'smooth' });
    } else if (direction === "down") {
      contentDiv.scrollBy({ top: scrollAmount, behavior: 'smooth' });
    }
  });

  EventsOn('solution-stream-start', () => {
    hasStarted.value = true
    handleStreamStart()
  })

  EventsOn('solution-stream-chunk', (token) => {
    handleStreamChunk(token)
  })

  EventsOn('solution-stream-thinking', (token) => {
    handleThinkingChunk(token)
  })

  // 错误处理
  EventsOn('solution-error', (rawErrMsg) => {
    // A. 优先处理：用户取消 (这不是错误，是操作)
    if (rawErrMsg && (rawErrMsg.includes('context canceled') || rawErrMsg.includes('canceled'))) {
      handleUserCancellation()
      return
    }

    // 直接显示上游返回的错误信息
    let title = '请求出错'
    let desc = rawErrMsg || '未知错误'
    let icon = '❌'

    // 尝试解析 JSON 格式的错误
    try {
      const errObj = JSON.parse(rawErrMsg)
      if (errObj.message) {
        desc = errObj.message
      }
      if (errObj.statusCode) {
        title = `错误 ${errObj.statusCode}`
      }
    } catch (e) {
      // 如果不是 JSON，直接使用原始字符串
    }

    // 更新 UI 状态
    statusText.value = '出错'
    statusIcon.value = '🔴'
    errorState.show = true
    errorState.title = title
    errorState.desc = desc
    errorState.icon = icon
    errorState.rawError = rawErrMsg
    errorState.showDetails = false
    isLoading.value = false
    isAppending.value = false
    shouldOverwriteHistory.value = true
  })

  // 抽离取消逻辑
  function handleUserCancellation() {
    console.log('请求已由用户主动取消')

    // 恢复状态
    if (isLoading.value) isLoading.value = true
    if (isAppending.value) isAppending.value = true

    // 回滚历史记录逻辑
    if (history.value.length > 0 && activeHistoryIndex.value === 0) {
      const current = history.value[0]

      if (settings.keepContext && current.rounds?.length > 1) {
        // 移除最后一轮（未完成的），Vue 响应式自动更新视图
        current.rounds.pop()
        setStreamBuffer('')

        isAppending.value = true
        isLoading.value = false
        shouldOverwriteHistory.value = false
      } else {
        // 不保留上下文或只有一轮，重置当前历史
        resetCurrentHistory(current)
        shouldOverwriteHistory.value = true
      }
    }
  }

  // 辅助函数
  function resetCurrentHistory(current) {
    if (current.rounds?.length) {
      current.rounds[0].aiResponse = ''
    }
    setStreamBuffer('')
    isLoading.value = true
    statusText.value = '正在思考...'
    statusIcon.value = '🟡'
  }

  EventsOn('require-login', () => {
    // 强制更新 settings 关闭 Live Mode
    if (settings.useLiveApi) settings.useLiveApi = false
    
    // 如果设置面板已经打开，直接更新 tempSettings 防止重置用户未保存的修改
    if (uiState.showSettings) {
      if (tempSettings.useLiveApi) tempSettings.useLiveApi = false
    } else {
      // 面板未打开，调用 openSettings 进行初始化
      openSettings()
    }
    
    uiState.activeTab = 'account' // 通过 v-model 同步到 SettingsModal
    showToast('请先配置 API Key', 'warning')
  })

  const mainInterface = document.getElementById('main-interface')
  if (mainInterface) mainInterface.style.pointerEvents = 'auto'

  // document.addEventListener('contextmenu', event => event.preventDefault());

  document.addEventListener('keydown', event => {
    if (
      event.key === 'F12' ||
      (event.ctrlKey && event.shiftKey && event.key === 'I') ||
      (event.ctrlKey && event.shiftKey && event.key === 'J') ||
      (event.ctrlKey && event.key === 'U')
    ) {
      event.preventDefault();
    }
  });
})
</script>
