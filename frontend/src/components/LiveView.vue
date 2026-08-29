<template>
  <div class="live-view">
    <!-- 顶部状态栏 -->
    <div class="live-header">
      <div class="header-left">
        <div class="audio-bars" :class="{ active: status === 'connected' && isSpeechActive }">
          <span></span><span></span><span></span>
        </div>
        <span class="title-text">实时助手</span>
        <div class="header-status" :class="status">
          <span class="status-dot"></span>
          <span>{{ statusText }}</span>
        </div>
      </div>
      <div class="header-right">
        <span class="duration">{{ sessionDuration }}</span>
        <button class="export-btn" @click="exportNotes" :disabled="treeNodes.length === 0">
          📤 导出
        </button>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="status === 'error' && errorMsg" class="error-banner">
      <span>⚠️ {{ errorMsg }}</span>
      <button @click="retryConnection">重试</button>
    </div>

    <!-- 主内容区 - 双栏布局 -->
    <div class="main-content">
      <!-- 左侧：实时对话流 -->
      <div class="chat-column">
        <div class="chat-area" ref="chatContainer">
          <div v-if="messages.length === 0" class="empty-state">
            <div class="empty-icon">🎙️</div>
            <div class="empty-title">准备就绪</div>
            <div class="empty-desc">开始说话，AI 将实时响应</div>
          </div>
          <template v-else>
            <div v-for="(msg, idx) in messages" :key="msg.id" 
                 class="msg-item" :class="[msg.type, { highlight: highlightMsgId === msg.id }]"
                 :id="'msg-' + msg.id">
              <div class="msg-round" v-if="msg.type === 'interviewer'">R{{ getRoundNumber(idx) }}</div>
              <div class="msg-content">
                <div class="msg-role">{{ msg.type === 'interviewer' ? '🎤 语音' : '🤖 AI' }}</div>
                <div class="msg-text" v-html="msg.type === 'ai' ? renderMarkdown(msg.content) : escapeHtml(msg.content)"></div>
                <div v-if="!msg.isComplete" class="typing-indicator">
                  <span></span><span></span><span></span>
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>

      <!-- 右侧：问题导图 + 详情 -->
      <div class="tree-column">
        <!-- 问题导图 -->
        <div class="tree-panel">
          <div class="panel-header">
            <div class="panel-title">🗺️ 问题导图</div>
            <div class="panel-controls">
              <button class="control-btn" @click="handleZoomOut" title="缩小">−</button>
              <button class="control-btn" @click="handleZoomIn" title="放大">+</button>
              <button class="control-btn" @click="fitView" title="适应视图">⟲</button>
            </div>
          </div>
          <div class="tree-container">
            <div v-show="flowNodes.length === 0" class="tree-empty">
              对话开始后自动生成
            </div>
            <VueFlow
              v-show="flowNodes.length > 0"
              :nodes="flowNodes"
              :edges="flowEdges"
              :node-types="nodeTypes"
              :default-viewport="{ x: 0, y: 0, zoom: 0.8 }"
              :min-zoom="0.2"
              :max-zoom="2"
              :fit-view-on-init="true"
              :nodes-draggable="false"
              :nodes-connectable="false"
              :edges-updatable="false"
              :pan-on-drag="true"
              :zoom-on-scroll="true"
              :zoom-on-pinch="true"
              :pan-on-scroll="false"
              @node-click="onNodeClick"
              class="question-flow"
            >
              <Background :gap="16" :size="1" pattern-color="rgba(255,255,255,0.03)" />
            </VueFlow>
          </div>
        </div>

        <!-- 节点详情 -->
        <div class="detail-panel">
          <div class="panel-title">📝 节点详情</div>
          <div v-if="!selectedNode" class="detail-empty">
            点击节点查看详情
          </div>
          <div v-else class="detail-content">
            <!-- 路径导航 -->
            <div class="detail-path">
              <span v-for="(p, i) in selectedNodePath" :key="p.id" class="path-item">
                {{ p.title }}
                <span v-if="i < selectedNodePath.length - 1" class="path-sep">→</span>
              </span>
            </div>
            
            <!-- 问题卡片 -->
            <div class="detail-card question-card">
              <div class="card-header">
                <span class="card-icon">❓</span>
                <span class="card-title">问题</span>
              </div>
              <div class="card-body">{{ selectedNode.question }}</div>
            </div>
            
            <!-- 解析卡片 -->
            <div class="detail-card answer-card">
              <div class="card-header">
                <span class="card-icon">💡</span>
                <span class="card-title">解析</span>
              </div>
              <div class="card-body markdown" v-html="renderMarkdown(selectedNode.answer)"></div>
            </div>
            
            <!-- 要点（如果有） -->
            <div v-if="selectedNode.keyPoints?.length" class="detail-card keypoints-card">
              <div class="card-header">
                <span class="card-icon">📌</span>
                <span class="card-title">要点</span>
              </div>
              <ul class="key-points">
                <li v-for="(point, i) in selectedNode.keyPoints" :key="i">{{ point }}</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick, markRaw } from 'vue'
import { marked } from 'marked'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { StartLiveSession, StopLiveSession, GetSettings } from '../../wailsjs/go/main/App'
import QuestionNode from './QuestionNode.vue'

// Vue Flow 节点类型
const nodeTypes = {
  question: markRaw(QuestionNode)
}

// Vue Flow 实例和方法
const { fitView: vfFitView, setCenter, getViewport, zoomIn, zoomOut } = useVueFlow()

function fitView() {
  nextTick(() => {
    vfFitView({ padding: 0.2 })
  })
}

function handleZoomIn() {
  zoomIn({ duration: 200 })
}

function handleZoomOut() {
  zoomOut({ duration: 200 })
}

// ===== 状态 =====
const status = ref('disconnected')
const errorMsg = ref('')
const chatContainer = ref(null)
const treeContainer = ref(null)
const messages = ref([])
const highlightMsgId = ref(null)
const voiceReplyEnabled = ref(false)
const isSpeechActive = ref(false)
let pendingSpeech = ''

// 问题导图数据
const treeNodes = ref([])  // 原始节点数据
const selectedNodeId = ref(null)

// 计时
const sessionStartTime = ref(Date.now())
const sessionDuration = ref('00:00')

// ===== 计算属性 =====
const statusText = computed(() => {
  const map = { disconnected: '未连接', connecting: '连接中...', connected: '已连接', error: '连接失败' }
  if (status.value === 'connected' && isSpeechActive.value) return '正在识别语音...'
  return map[status.value] || '未知'
})

const selectedNode = computed(() => treeNodes.value.find(n => n.id === selectedNodeId.value))

const selectedNodePath = computed(() => {
  if (!selectedNode.value) return []
  const path = []
  let current = selectedNode.value
  while (current) {
    path.unshift(current)
    current = treeNodes.value.find(n => n.id === current.pid)
  }
  return path
})

// 选中节点的完整路径 ID 集合（用于高亮）
const selectedPathIds = computed(() => {
  return new Set(selectedNodePath.value.map(n => n.id))
})

// 选中路径上的边 ID 集合
const selectedPathEdgeIds = computed(() => {
  const edgeIds = new Set()
  const path = selectedNodePath.value
  for (let i = 1; i < path.length; i++) {
    edgeIds.add(`e-${path[i-1].id}-${path[i].id}`)
  }
  return edgeIds
})

// Vue Flow 节点（从 treeNodes 转换）
const flowNodes = computed(() => {
  if (treeNodes.value.length === 0) return []
  
  // 构建层级结构
  const nodeMap = {}
  const levels = {}
  
  treeNodes.value.forEach((n, idx) => {
    nodeMap[n.id] = { ...n, index: idx + 1, children: [] }
  })
  
  treeNodes.value.forEach(n => {
    if (n.pid && nodeMap[n.pid]) {
      nodeMap[n.pid].children.push(nodeMap[n.id])
    }
  })
  
  // 计算层级
  function assignLevel(node, level) {
    if (!levels[level]) levels[level] = []
    levels[level].push(node)
    node.level = level
    node.children.forEach(c => assignLevel(c, level + 1))
  }
  
  const roots = treeNodes.value.filter(n => !n.pid)
  roots.forEach(r => assignLevel(nodeMap[r.id], 0))
  
  // 生成 Vue Flow 节点
  const nodes = []
  const nodeSpacingX = 180  // 水平间距
  const nodeSpacingY = 120  // 垂直间距
  Object.keys(levels).forEach(level => {
    const levelNodes = levels[level]
    const y = parseInt(level) * nodeSpacingY + 50
    const totalWidth = levelNodes.length * nodeSpacingX
    const startX = -totalWidth / 2 + nodeSpacingX / 2
    
    levelNodes.forEach((node, i) => {
      nodes.push({
        id: node.id,
        type: 'question',
        position: { x: startX + i * nodeSpacingX, y },
        data: {
          title: truncate(node.title, 8),
          index: node.index,
          selected: selectedNodeId.value === node.id,
          inPath: selectedPathIds.value.has(node.id) && selectedNodeId.value !== node.id
        }
      })
    })
  })
  
  return nodes
})

// Vue Flow 边（连接线）
const flowEdges = computed(() => {
  // 获取所有有效节点ID
  const validNodeIds = new Set(treeNodes.value.map(n => n.id))
  
  return treeNodes.value
    .filter(n => n.pid && validNodeIds.has(n.pid)) // 确保 pid 指向的节点存在
    .map(n => {
      const edgeId = `e-${n.pid}-${n.id}`
      const isInPath = selectedPathEdgeIds.value.has(edgeId)
      
      return {
        id: edgeId,
        source: n.pid,
        target: n.id,
        type: 'smoothstep',
        animated: isInPath,
        style: {
          stroke: isInPath ? '#10b981' : 'rgba(99, 102, 241, 0.5)',
          strokeWidth: isInPath ? 2.5 : 1.5,
          opacity: isInPath ? 1 : 0.6
        }
      }
    })
})

// ===== 方法 =====
function generateId() {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 6)}`
}

function escapeHtml(text) {
  if (!text) return ''
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>')
}

function renderMarkdown(text) {
  if (!text) return ''
  return marked.parse(text.replace(/\n+$/, ''))
}

function truncate(str, len) {
  if (!str) return ''
  return str.length > len ? str.slice(0, len) + '...' : str
}

function getRoundNumber(msgIndex) {
  let round = 0
  for (let i = 0; i <= msgIndex; i++) {
    if (messages.value[i]?.type === 'interviewer') round++
  }
  return round
}

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

// Vue Flow 节点点击
function onNodeClick({ node }) {
  selectedNodeId.value = node.id
  const treeNode = treeNodes.value.find(n => n.id === node.id)
  if (treeNode?.msgId) {
    highlightMsgId.value = treeNode.msgId
    const el = document.getElementById('msg-' + treeNode.msgId)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
    setTimeout(() => highlightMsgId.value = null, 2000)
  }
}

// ===== 后端调用接口（通过事件） =====
// 后端可以通过 EventsEmit 调用这些方法来操作导图

/**
 * 添加节点（供后端调用）
 * @param {Object} data - { id?, pid?, title, question, answer, keyPoints?, msgId? }
 */
function addNodeFromBackend(data) {
  const node = {
    id: data.id || generateId(),
    pid: data.pid || (treeNodes.value.length > 0 ? treeNodes.value[treeNodes.value.length - 1].id : null),
    title: data.title || data.question?.slice(0, 20) || '未知问题',
    question: data.question || '',
    answer: data.answer || '',
    msgId: data.msgId || null,
    keyPoints: data.keyPoints || [],
    timestamp: Date.now()
  }
  treeNodes.value.push(node)
  selectedNodeId.value = node.id
  
  // 自动跳转到新节点位置
  nextTick(() => {
    focusOnNode(node.id)
  })
}

/**
 * 跳转到指定节点，保持当前缩放，居中显示
 * @param {string} nodeId - 节点ID
 */
function focusOnNode(nodeId) {
  const flowNode = flowNodes.value.find(n => n.id === nodeId)
  if (!flowNode) return

  // 保持当前缩放级别，只平移让节点居中
  const currentZoom = getViewport().zoom
  setCenter(flowNode.position.x, flowNode.position.y, {
    zoom: currentZoom,
    duration: 400
  })
}

/**
 * 更新节点（供后端调用）
 * @param {Object} data - { id, title?, question?, answer?, keyPoints? }
 */
function updateNodeFromBackend(data) {
  const node = treeNodes.value.find(n => n.id === data.id)
  if (node) {
    if (data.title !== undefined) node.title = data.title
    if (data.question !== undefined) node.question = data.question
    if (data.answer !== undefined) node.answer = data.answer
    if (data.keyPoints !== undefined) node.keyPoints = data.keyPoints
  }
}

/**
 * 清空导图（供后端调用）
 */
function clearNodesFromBackend() {
  treeNodes.value = []
  selectedNodeId.value = null
}

/**
 * 删除节点（供后端调用）
 * @param {Object} data - { id }
 */
function removeNodeFromBackend(data) {
  const idx = treeNodes.value.findIndex(n => n.id === data.id)
  if (idx !== -1) {
    // 同时删除该节点的所有子节点
    const idsToRemove = new Set([data.id])
    let changed = true
    while (changed) {
      changed = false
      treeNodes.value.forEach(n => {
        if (n.pid && idsToRemove.has(n.pid) && !idsToRemove.has(n.id)) {
          idsToRemove.add(n.id)
          changed = true
        }
      })
    }
    treeNodes.value = treeNodes.value.filter(n => !idsToRemove.has(n.id))
    if (idsToRemove.has(selectedNodeId.value)) {
      selectedNodeId.value = null
    }
  }
}

function exportNotes() {
  if (treeNodes.value.length === 0) return
  
  const now = new Date()
  const dateStr = now.toLocaleString('zh-CN')
  
  let md = `# 面试笔记\n\n`
  md += `**时间**: ${dateStr}\n`
  md += `**时长**: ${sessionDuration.value}\n\n`
  md += `---\n\n`
  md += `## 问题导图\n\n`
  
  // 构建树形 markdown
  function renderTree(node, indent = '') {
    let result = `${indent}- **${node.title}**\n`
    if (node.keyPoints?.length) {
      node.keyPoints.forEach(p => {
        result += `${indent}  - ${p}\n`
      })
    }
    const children = treeNodes.value.filter(n => n.pid === node.id)
    children.forEach(c => {
      result += renderTree(c, indent + '  ')
    })
    return result
  }
  
  const roots = treeNodes.value.filter(n => !n.pid)
  roots.forEach(r => {
    md += renderTree(r)
  })
  
  md += `\n---\n\n## 完整对话\n\n`
  
  let roundNum = 0
  messages.value.forEach(msg => {
    if (msg.type === 'interviewer') {
      roundNum++
      md += `### Round ${roundNum}\n\n`
      md += `**Q**: ${msg.content}\n\n`
    } else {
      md += `**A**: ${msg.content}\n\n`
    }
  })
  
  // 下载文件
  const blob = new Blob([md], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `面试笔记_${now.toISOString().slice(0, 10)}.md`
  a.click()
  URL.revokeObjectURL(url)
}

// ===== 事件处理 =====
function onLiveStatus(s) {
  status.value = s
  if (s === 'connected') {
    sessionStartTime.value = Date.now()
    startTimer()
  } else if (s === 'error' || s === 'disconnected') {
    stopTimer()
  }
}

function onLiveTranscript(text) {
  const lastMsg = messages.value[messages.value.length - 1]
  
  // 结束上一条 AI 消息
  if (lastMsg?.type === 'ai' && !lastMsg.isComplete) {
    lastMsg.isComplete = true
  }
  
  // 追加或新建语音消息
  if (lastMsg?.type === 'interviewer' && !lastMsg.isComplete) {
    lastMsg.content += text
  } else {
    const newMsg = { id: generateId(), type: 'interviewer', content: text, timestamp: Date.now(), isComplete: false }
    messages.value.push(newMsg)
  }
  scrollToBottom()
}

// Qwen transcription delta 的 text + stash 是完整预览，必须替换而不是追加。
function onLiveTranscriptPreview(data) {
  const itemId = data?.itemId || 'qwen-current'
  const text = data?.text || ''
  let msg = messages.value.find(message => message.type === 'interviewer' && message.itemId === itemId)
  if (!msg) {
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg?.type === 'ai' && !lastMsg.isComplete) lastMsg.isComplete = true
    msg = { id: generateId(), itemId, type: 'interviewer', content: text, timestamp: Date.now(), isComplete: false, isPreview: true }
    messages.value.push(msg)
  } else {
    msg.content = text
    msg.isPreview = true
  }
  scrollToBottom()
}

// completed.transcript 才是最终问题；按 item_id 精确替换并只保留一份。
function onLiveTranscriptFinal(data) {
  const itemId = data?.itemId || 'qwen-current'
  const text = data?.text || ''
  let msg = messages.value.find(message => message.type === 'interviewer' && message.itemId === itemId)
  if (!msg) {
    msg = { id: generateId(), itemId, type: 'interviewer', content: text, timestamp: Date.now(), isComplete: true }
    messages.value.push(msg)
  } else {
    msg.content = text
    msg.isPreview = false
    msg.isComplete = true
  }
  scrollToBottom()
}

function onLiveAiText(text) {
  const lastMsg = messages.value[messages.value.length - 1]
  
  // 结束上一条语音消息
  if (lastMsg?.type === 'interviewer' && !lastMsg.isComplete) {
    lastMsg.isComplete = true
  }
  
  // 追加或新建 AI 消息
  if (lastMsg?.type === 'ai' && !lastMsg.isComplete) {
    lastMsg.content += text
  } else {
    const newMsg = { id: generateId(), type: 'ai', content: text, timestamp: Date.now(), isComplete: false }
    messages.value.push(newMsg)
  }
  scrollToBottom()
	if (voiceReplyEnabled.value) pendingSpeech += text
}

function onLiveError(err) {
  status.value = 'error'
  errorMsg.value = err
}

function onLiveDone() {
  const lastMsg = messages.value[messages.value.length - 1]
  if (lastMsg) lastMsg.isComplete = true
	if (voiceReplyEnabled.value && pendingSpeech.trim() && 'speechSynthesis' in window) {
		window.speechSynthesis.cancel()
		const utterance = new SpeechSynthesisUtterance(pendingSpeech.replace(/[`#*_>-]/g, ' '))
		utterance.lang = 'zh-CN'
		window.speechSynthesis.speak(utterance)
	}
	pendingSpeech = ''
}

function onLiveInterrupted() {
  const lastMsg = messages.value[messages.value.length - 1]
  if (lastMsg?.type === 'ai') {
    lastMsg.isComplete = true
    lastMsg.interrupted = true
  }
}

function onSpeechStarted() {
  isSpeechActive.value = true
}

function onSpeechStopped() {
  isSpeechActive.value = false
}

function retryConnection() {
  errorMsg.value = ''
  status.value = 'connecting'
  StopLiveSession()
  StartLiveSession().catch(err => onLiveError(err?.message || String(err)))
}

// 计时器
let timer = null
function startTimer() {
  timer = setInterval(() => {
    const elapsed = Math.floor((Date.now() - sessionStartTime.value) / 1000)
    const m = Math.floor(elapsed / 60).toString().padStart(2, '0')
    const s = (elapsed % 60).toString().padStart(2, '0')
    sessionDuration.value = `${m}:${s}`
  }, 1000)
}

function stopTimer() {
  if (timer) clearInterval(timer)
}

// ===== 生命周期 =====
onMounted(async () => {
	try {
		const cfg = await GetSettings()
		voiceReplyEnabled.value = !!(cfg.sttEnabled && cfg.voiceReply)
	} catch (e) {
		console.warn('读取语音回复设置失败', e)
	}
  // Live API 事件
  EventsOn('live:status', onLiveStatus)
  EventsOn('live:transcript', onLiveTranscript)
  EventsOn('live:transcript-preview', onLiveTranscriptPreview)
  EventsOn('live:transcript-final', onLiveTranscriptFinal)
  EventsOn('live:ai-text', onLiveAiText)
  EventsOn('live:error', onLiveError)
  EventsOn('live:done', onLiveDone)
  EventsOn('live:speech-started', onSpeechStarted)
  EventsOn('live:speech-stopped', onSpeechStopped)
  EventsOn('live:Interrupted', onLiveInterrupted)
  
  // 导图节点操作事件（供后端调用）
  EventsOn('graph:add-node', addNodeFromBackend)
  EventsOn('graph:update-node', updateNodeFromBackend)
  EventsOn('graph:remove-node', removeNodeFromBackend)
  EventsOn('graph:clear', clearNodesFromBackend)
  
  StartLiveSession().catch(err => onLiveError(err?.message || String(err)))
})

onUnmounted(() => {
	if ('speechSynthesis' in window) window.speechSynthesis.cancel()
  StopLiveSession()
  stopTimer()
  
  // 移除 Live API 事件
  EventsOff('live:status')
  EventsOff('live:transcript')
  EventsOff('live:transcript-preview')
  EventsOff('live:transcript-final')
  EventsOff('live:ai-text')
  EventsOff('live:error')
  EventsOff('live:done')
  EventsOff('live:speech-started')
  EventsOff('live:speech-stopped')
  EventsOff('live:Interrupted')
  
  // 移除导图事件
  EventsOff('graph:add-node')
  EventsOff('graph:update-node')
  EventsOff('graph:remove-node')
  EventsOff('graph:clear')
})

watch(messages, scrollToBottom, { deep: true })
</script>

<style scoped>
/* ===== 基础布局 ===== */
.live-view {
  flex: 1;  /* 占据剩余空间 */
  display: flex;
  flex-direction: column;
  min-height: 0;  /* 重要：允许flex子项收缩 */
  overflow: hidden;  /* 防止整体滚动 */
  pointer-events: auto;
  user-select: none;  /* 禁止文本选择 */
  -webkit-user-select: none;
}

/* ===== 顶部栏 ===== */
.live-header {
  flex-shrink: 0;  /* 不收缩 */
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.header-left, .header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.audio-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 14px;
}

.audio-bars span {
  width: 3px;
  height: 4px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 2px;
}

.audio-bars.active span {
  background: #10b981;
  animation: wave 0.8s ease-in-out infinite;
}

.audio-bars.active span:nth-child(2) { animation-delay: 0.2s; }
.audio-bars.active span:nth-child(3) { animation-delay: 0.4s; }

@keyframes wave {
  0%, 100% { height: 4px; }
  50% { height: 14px; }
}

.title-text {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
}

.header-status {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.5);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
}

.header-status.connected { color: #10b981; }
.header-status.connected .status-dot { background: #10b981; box-shadow: 0 0 6px #10b981; }
.header-status.error { color: #ef4444; }
.header-status.error .status-dot { background: #ef4444; }

.duration {
  font-size: 12px;
  font-family: monospace;
  color: rgba(255, 255, 255, 0.6);
}

.export-btn {
  padding: 5px 12px;
  font-size: 11px;
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 6px;
  color: #10b981;
  cursor: pointer;
  transition: all 0.2s;
}

.export-btn:hover:not(:disabled) {
  background: rgba(16, 185, 129, 0.25);
}

.export-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ===== 错误提示 ===== */
.error-banner {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 8px 16px;
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  font-size: 12px;
  color: #fca5a5;
}

.error-banner button {
  padding: 4px 12px;
  background: rgba(239, 68, 68, 0.2);
  border: 1px solid rgba(239, 68, 68, 0.4);
  border-radius: 4px;
  color: #fca5a5;
  cursor: pointer;
}

/* ===== 主内容区 ===== */
.main-content {
  display: flex;
  flex: 1;
  gap: 12px;
  padding: 12px 16px;
  min-height: 0;  /* 重要：允许flex子项收缩 */
  overflow: hidden;
}

/* ===== 左侧对话栏 ===== */
.chat-column {
  flex: 1;
  min-width: 0;
  min-height: 0;  /* 重要 */
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-area {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 8px;
  padding-bottom: 24px;  /* 底部留出间距 */
}

.chat-area::-webkit-scrollbar { width: 4px; }
.chat-area::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); border-radius: 2px; }

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.5);
}

.empty-icon { font-size: 36px; }
.empty-title { font-size: 16px; font-weight: 600; color: rgba(255, 255, 255, 0.8); }
.empty-desc { font-size: 13px; }

/* 消息项 */
.msg-item {
  display: flex;
  gap: 8px;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.msg-item.highlight .msg-content {
  box-shadow: 0 0 0 2px #10b981;
}

.msg-round {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(139, 92, 246, 0.2);
  border-radius: 6px;
  font-size: 10px;
  font-weight: 700;
  color: #a78bfa;
}

.msg-item.ai .msg-round { display: none; }

.msg-content {
  flex: 1;
  padding: 10px 12px;
  border-radius: 10px;
  transition: box-shadow 0.3s;
}

.msg-item.interviewer .msg-content {
  background: rgba(70, 80, 100, 0.5);
  border-left: 3px solid #8b5cf6;
}

.msg-item.ai .msg-content {
  background: rgba(16, 185, 129, 0.12);
  border-left: 3px solid #10b981;
}

.msg-role {
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 6px;
  color: rgba(255, 255, 255, 0.6);
}

.msg-text {
  font-size: 13px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.9);
}

.msg-text :deep(p) { margin: 0 0 8px; }
.msg-text :deep(p:last-child) { margin: 0; }
.msg-text :deep(code) {
  background: rgba(0, 0, 0, 0.3);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.typing-indicator {
  display: flex;
  gap: 4px;
  margin-top: 8px;
}

.typing-indicator span {
  width: 5px;
  height: 5px;
  background: rgba(255, 255, 255, 0.4);
  border-radius: 50%;
  animation: bounce 1.2s infinite;
}

.typing-indicator span:nth-child(2) { animation-delay: 0.15s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.3s; }

@keyframes bounce {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-5px); }
}

/* ===== 右侧树栏 ===== */
.tree-column {
  width: 380px;
  flex-shrink: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.panel-title {
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.6);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.panel-controls {
  display: flex;
  gap: 4px;
}

.control-btn {
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  border: none;
  border-radius: 4px;
  color: rgba(255, 255, 255, 0.5);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.8);
}

/* 问题导图面板 */
.tree-panel {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 12px;
  flex: 1;
  min-height: 280px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tree-container {
  flex: 1;
  overflow: hidden;
  min-height: 0;
  position: relative;
  border-radius: 6px;
}

.tree-empty {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  cursor: default;
}

/* Vue Flow 样式覆盖 */
.question-flow {
  width: 100%;
  height: 100%;
}

.question-flow :deep(.vue-flow__pane) {
  cursor: grab;
}

.question-flow :deep(.vue-flow__pane:active) {
  cursor: grabbing;
}

/* 边的入场动画 */
@keyframes edgeFadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.question-flow :deep(.vue-flow__edge-path) {
  stroke: rgba(99, 102, 241, 0.5);
  stroke-width: 2;
  animation: edgeFadeIn 0.4s ease-out forwards;
  animation-delay: 0.3s;
  opacity: 0;
}

.question-flow :deep(.vue-flow__edge.animated .vue-flow__edge-path) {
  stroke: #10b981;
  stroke-dasharray: 5;
  animation: dashdraw 0.5s linear infinite;
  opacity: 1;
}

@keyframes dashdraw {
  from { stroke-dashoffset: 10; }
  to { stroke-dashoffset: 0; }
}

.question-flow :deep(.vue-flow__background) {
  background: transparent;
}

/* 详情面板 */
.detail-panel {
  flex-shrink: 0;
  max-height: 200px;
  min-height: 120px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 12px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.detail-panel::-webkit-scrollbar { width: 4px; }
.detail-panel::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); border-radius: 2px; }

.detail-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-path {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 10px;
  color: rgba(255, 255, 255, 0.5);
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.path-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.path-sep { color: #10b981; }

/* 详情卡片 */
.detail-card {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  font-size: 11px;
  font-weight: 600;
}

.card-icon {
  font-size: 12px;
}

.card-title {
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.card-body {
  padding: 10px;
  font-size: 12px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.85);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

/* 问题卡片 */
.question-card .card-header {
  background: rgba(139, 92, 246, 0.15);
  color: #a78bfa;
}

.question-card .card-body {
  background: rgba(139, 92, 246, 0.05);
}

/* 解析卡片 */
.answer-card .card-header {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}

.answer-card .card-body {
  background: rgba(16, 185, 129, 0.05);
}

.card-body.markdown :deep(p) { margin: 0 0 8px; }
.card-body.markdown :deep(p:last-child) { margin: 0; }
.card-body.markdown :deep(code) {
  background: rgba(0, 0, 0, 0.3);
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
}
.card-body.markdown :deep(ul),
.card-body.markdown :deep(ol) {
  margin: 4px 0;
  padding-left: 18px;
}

/* 要点卡片 */
.keypoints-card .card-header {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}

.key-points {
  margin: 0;
  padding: 10px 10px 10px 26px;
  font-size: 11px;
  line-height: 1.8;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(251, 191, 36, 0.05);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.key-points li {
  margin-bottom: 2px;
}
.key-points li::marker {
  color: #fbbf24;
}

/* ===== 白天主题 ===== */
:global(html[data-theme="light"]) .live-header {
  border-bottom-color: rgba(15, 23, 42, 0.1);
}

:global(html[data-theme="light"]) .audio-bars span,
:global(html[data-theme="light"]) .status-dot {
  background: rgba(15, 23, 42, 0.24);
}

:global(html[data-theme="light"]) .title-text,
:global(html[data-theme="light"]) .empty-title {
  color: #0f172a;
}

:global(html[data-theme="light"]) .header-status,
:global(html[data-theme="light"]) .duration,
:global(html[data-theme="light"]) .empty-state,
:global(html[data-theme="light"]) .msg-role,
:global(html[data-theme="light"]) .panel-title,
:global(html[data-theme="light"]) .detail-path {
  color: #64748b;
}

:global(html[data-theme="light"]) .error-banner {
  color: #b91c1c;
  background: rgba(254, 226, 226, 0.86);
  border-color: rgba(220, 38, 38, 0.22);
}

:global(html[data-theme="light"]) .error-banner button {
  color: #b91c1c;
  background: rgba(220, 38, 38, 0.08);
  border-color: rgba(220, 38, 38, 0.2);
}

:global(html[data-theme="light"]) .chat-area::-webkit-scrollbar-thumb,
:global(html[data-theme="light"]) .detail-panel::-webkit-scrollbar-thumb {
  background: rgba(15, 23, 42, 0.16);
}

:global(html[data-theme="light"]) .msg-item.interviewer .msg-content {
  background: rgba(139, 92, 246, 0.09);
}

:global(html[data-theme="light"]) .msg-item.ai .msg-content {
  background: rgba(5, 150, 105, 0.08);
}

:global(html[data-theme="light"]) .msg-text,
:global(html[data-theme="light"]) .card-body,
:global(html[data-theme="light"]) .key-points {
  color: #1e293b;
}

:global(html[data-theme="light"]) .msg-text :deep(code),
:global(html[data-theme="light"]) .card-body.markdown :deep(code) {
  color: #0f172a;
  background: rgba(15, 23, 42, 0.07);
}

:global(html[data-theme="light"]) .typing-indicator span {
  background: rgba(15, 23, 42, 0.32);
}

:global(html[data-theme="light"]) .control-btn {
  color: #64748b;
  background: rgba(15, 23, 42, 0.06);
}

:global(html[data-theme="light"]) .control-btn:hover {
  color: #1e293b;
  background: rgba(15, 23, 42, 0.1);
}

:global(html[data-theme="light"]) .tree-panel,
:global(html[data-theme="light"]) .detail-panel {
  background: rgba(255, 255, 255, 0.58);
  border-color: rgba(15, 23, 42, 0.1);
}

:global(html[data-theme="light"]) .tree-empty,
:global(html[data-theme="light"]) .detail-empty {
  color: rgba(71, 85, 105, 0.7);
}

:global(html[data-theme="light"]) .detail-path {
  border-bottom-color: rgba(15, 23, 42, 0.1);
}

:global(html[data-theme="light"]) .detail-card {
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(15, 23, 42, 0.07);
}

:global(html[data-theme="light"]) .card-body,
:global(html[data-theme="light"]) .key-points {
  border-top-color: rgba(15, 23, 42, 0.07);
}

:global(html[data-theme="light"]) .question-card .card-body {
  background: rgba(139, 92, 246, 0.045);
}

:global(html[data-theme="light"]) .answer-card .card-body {
  background: rgba(5, 150, 105, 0.045);
}

:global(html[data-theme="light"]) .answer-card .card-header {
  color: #047857;
}

:global(html[data-theme="light"]) .key-points {
  background: rgba(245, 158, 11, 0.06);
}
</style>
