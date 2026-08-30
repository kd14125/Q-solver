<template>
  <section class="rag-panel">
    <div class="setting-row rag-heading">
      <div class="setting-info">
        <span class="setting-title">面试知识库（RAG）</span>
        <span class="setting-desc">仅用于 Qwen Realtime；截图答题和第三方 STT 不受影响</span>
      </div>
      <label class="switch"><input type="checkbox" v-model="settings.ragEnabled"><span class="slider round"></span></label>
    </div>

    <template v-if="settings.ragEnabled">
      <label class="field-label">检索方式</label>
      <div class="mode-grid">
        <button v-for="mode in modes" :key="mode.value" type="button" class="mode-card"
          :class="{ active: settings.ragRetrievalMode === mode.value }"
          @click="settings.ragRetrievalMode = mode.value">
          <strong>{{ mode.label }}</strong><small>{{ mode.description }}</small>
        </button>
      </div>

      <p class="inherit-hint" v-if="settings.ragRetrievalMode !== 'local'">
        默认沿用当前语音 Realtime 的 API Key、Workspace ID 和地域；不会读取截图模型配置。
      </p>

      <button v-if="settings.ragRetrievalMode !== 'local'" type="button" class="text-button" @click="showAdvanced = !showAdvanced">
        {{ showAdvanced ? '收起' : '展开' }} Embedding 高级设置
      </button>
      <div v-if="showAdvanced && settings.ragRetrievalMode !== 'local'" class="advanced-grid">
        <label>Embedding 模型<input class="form-input" v-model.trim="settings.ragEmbeddingModel" placeholder="qwen3.7-text-embedding"></label>
        <label>向量维度<select class="form-input" v-model.number="settings.ragEmbeddingDimensions"><option v-for="dimension in [64,128,256,512,768,1024,1536,2048,2560]" :key="dimension" :value="dimension">{{ dimension }}</option></select></label>
        <label>RAG API Key（可选）<input class="form-input" type="password" autocomplete="off" v-model="settings.ragAPIKey" placeholder="留空沿用 Realtime API Key"></label>
        <label>RAG Workspace ID（可选）<input class="form-input" v-model.trim="settings.ragWorkspaceID" placeholder="留空沿用 Realtime Workspace ID"></label>
        <label>RAG 地域（可选）<input class="form-input" v-model.trim="settings.ragRegion" placeholder="留空沿用 Realtime 地域"></label>
        <label>RAG Base URL（可选）<input class="form-input" v-model.trim="settings.ragBaseURL" placeholder="自动生成 compatible-mode/v1 地址"></label>
        <label>返回条数<input class="form-input" type="number" min="1" max="20" v-model.number="settings.ragTopK"></label>
        <label>最大上下文字符<input class="form-input" type="number" min="500" max="30000" step="500" v-model.number="settings.ragMaxContextChars"></label>
        <button type="button" class="secondary-button" :disabled="busy.embedding" @click="testEmbedding">
          {{ busy.embedding ? '测试中…' : '测试 Embedding API' }}
        </button>
      </div>

      <div v-if="status.message" class="rag-status" :class="status.type">{{ status.message }}</div>

      <div class="toolbar">
        <button type="button" class="primary-button" :disabled="busy.importing" @click="importFiles">
          {{ busy.importing ? '导入和索引中…' : '导入 PDF / DOCX / TXT / Markdown' }}
        </button>
        <button type="button" class="secondary-button" :disabled="busy.rebuilding || settings.ragRetrievalMode === 'local'" @click="rebuildIndex">
          {{ busy.rebuilding ? '重建中…' : '重建向量索引' }}
        </button>
        <button type="button" class="secondary-button" :disabled="busy.loading" @click="loadKnowledge">刷新</button>
      </div>

      <div class="knowledge-list">
        <div class="list-title">已导入资料（{{ snapshot.documents.length }}）</div>
        <div v-if="!snapshot.documents.length" class="empty-note">尚未导入资料。</div>
        <div v-for="doc in snapshot.documents" :key="doc.id" class="knowledge-row">
          <div><strong>{{ doc.name }}</strong><small>{{ doc.kind.toUpperCase() }} · {{ doc.chunkCount }} 个片段 · {{ statusLabel(doc.status) }}</small></div>
          <button type="button" class="danger-button" @click="deleteDocument(doc.id)">删除</button>
        </div>
      </div>

      <div class="qa-editor">
        <div class="list-title">手动面试问答</div>
        <input class="form-input" v-model.trim="qa.question" placeholder="常见面试问题">
        <textarea class="form-input" rows="4" v-model.trim="qa.answer" placeholder="候选人可参考的事实或答案"></textarea>
        <div class="toolbar compact">
          <button type="button" class="primary-button" :disabled="busy.qa || !qa.question || !qa.answer" @click="saveQA">
            {{ qa.id ? '保存修改' : '添加问答' }}
          </button>
          <button v-if="qa.id" type="button" class="secondary-button" @click="resetQA">取消编辑</button>
        </div>
        <div v-for="entry in snapshot.qaEntries" :key="entry.id" class="qa-row">
          <div><strong>{{ entry.question }}</strong><p>{{ entry.answer }}</p><small>{{ statusLabel(entry.status) }}</small></div>
          <div class="row-actions"><button type="button" class="text-button" @click="editQA(entry)">编辑</button><button type="button" class="danger-button" @click="deleteQA(entry.id)">删除</button></div>
        </div>
      </div>

      <div class="search-test">
        <div class="list-title">测试检索效果</div>
        <div class="search-line"><input class="form-input" v-model.trim="testQuery" @keyup.enter="testSearch" placeholder="输入一个面试问题"><button type="button" class="primary-button" :disabled="busy.search || !testQuery" @click="testSearch">{{ busy.search ? '检索中…' : '测试' }}</button></div>
        <div v-if="testResult" class="result-grid">
          <div v-for="group in resultGroups" :key="group.key" class="result-group">
            <strong>{{ group.label }}</strong><small v-if="testResult[group.key]?.warning" class="warning-text">{{ testResult[group.key].warning }}</small>
            <div v-if="!testResult[group.key]?.hits?.length" class="empty-note">无命中</div>
            <div v-for="hit in testResult[group.key]?.hits || []" :key="`${group.key}-${hit.id}`" class="result-hit"><b>{{ hit.title }}</b><span>{{ hit.content }}</span></div>
          </div>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import {
  AddRAGQA, DeleteRAGDocument, DeleteRAGQA, GetRAGKnowledge, ImportRAGFiles,
  RebuildRAGIndex, TestRAGEmbedding, TestRAGSearch, UpdateRAGQA
} from '../../wailsjs/go/main/App'

const props = defineProps({ settings: { type: Object, required: true } })
const settings = props.settings
const modes = [
  { value: 'local', label: '本地检索', description: '中文关键词与 BM25，不发送资料' },
  { value: 'api', label: 'API 向量检索', description: '百炼 Embedding 语义召回' },
  { value: 'hybrid', label: '混合检索（推荐）', description: '同时使用关键词和语义向量' }
]
const resultGroups = [{ key: 'local', label: '本地检索' }, { key: 'api', label: 'API 向量检索' }, { key: 'hybrid', label: '混合结果' }]
const showAdvanced = ref(false)
const snapshot = reactive({ documents: [], qaEntries: [] })
const qa = reactive({ id: 0, question: '', answer: '' })
const testQuery = ref('')
const testResult = ref(null)
const status = reactive({ type: '', message: '' })
const busy = reactive({ loading: false, importing: false, rebuilding: false, embedding: false, qa: false, search: false })

function configJSON() {
  return JSON.stringify({
    realtimeAPIKey: settings.realtimeAPIKey,
    realtimeWorkspaceID: settings.realtimeWorkspaceID,
    realtimeRegion: settings.realtimeRegion,
    ragEnabled: settings.ragEnabled,
    ragRetrievalMode: settings.ragRetrievalMode,
    ragEmbeddingModel: settings.ragEmbeddingModel,
    ragEmbeddingDimensions: settings.ragEmbeddingDimensions,
    ragAPIKey: settings.ragAPIKey,
    ragWorkspaceID: settings.ragWorkspaceID,
    ragRegion: settings.ragRegion,
    ragBaseURL: settings.ragBaseURL,
    ragTopK: settings.ragTopK,
    ragMaxContextChars: settings.ragMaxContextChars
  })
}

function setStatus(type, message) { status.type = type; status.message = message }
function statusLabel(value) { return value === 'ready' ? '向量已就绪' : value === 'pending' ? '待生成向量' : '本地索引已就绪' }

async function loadKnowledge() {
  busy.loading = true
  try {
    const data = await GetRAGKnowledge(configJSON())
    snapshot.documents = data?.documents || []
    snapshot.qaEntries = data?.qaEntries || []
  } catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.loading = false }
}

async function testEmbedding() {
  busy.embedding = true
  try { const error = await TestRAGEmbedding(configJSON()); setStatus(error ? 'error' : 'success', error || 'Embedding API 连接成功') }
  catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.embedding = false }
}

async function importFiles() {
  busy.importing = true
  try {
    const results = await ImportRAGFiles(configJSON())
    const warnings = (results || []).filter(item => item.warning).map(item => item.warning)
    setStatus(warnings.length ? 'warning' : 'success', warnings.length ? `资料已导入，但有 ${warnings.length} 项索引警告：${warnings[0]}` : `已处理 ${(results || []).length} 个文件`)
    await loadKnowledge()
  } catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.importing = false }
}

async function rebuildIndex() {
  busy.rebuilding = true
  try { const result = await RebuildRAGIndex(configJSON()); setStatus(result?.warning ? 'warning' : 'success', result?.warning || `已生成 ${result?.indexed || 0}/${result?.total || 0} 条向量`); await loadKnowledge() }
  catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.rebuilding = false }
}

async function saveQA() {
  busy.qa = true
  try {
    let entry = null
    if (qa.id) await UpdateRAGQA(qa.id, qa.question, qa.answer, configJSON())
    else entry = await AddRAGQA(qa.question, qa.answer, configJSON())
    resetQA(); await loadKnowledge(); setStatus(entry?.warning ? 'warning' : 'success', entry?.warning || '问答条目已保存')
  } catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.qa = false }
}
function editQA(entry) { qa.id = entry.id; qa.question = entry.question; qa.answer = entry.answer }
function resetQA() { qa.id = 0; qa.question = ''; qa.answer = '' }
async function deleteQA(id) { try { await DeleteRAGQA(id); await loadKnowledge() } catch (error) { setStatus('error', error?.message || String(error)) } }
async function deleteDocument(id) { try { await DeleteRAGDocument(id); await loadKnowledge() } catch (error) { setStatus('error', error?.message || String(error)) } }

async function testSearch() {
  busy.search = true
  try { testResult.value = await TestRAGSearch(testQuery.value, configJSON()); setStatus('success', '检索测试完成') }
  catch (error) { setStatus('error', error?.message || String(error)) }
  finally { busy.search = false }
}

watch(() => settings.ragRetrievalMode, async (newMode, oldMode) => {
  if (oldMode === 'local' && newMode !== 'local' && settings.ragEnabled) await rebuildIndex()
})
onMounted(loadKnowledge)
</script>

<style scoped>
.rag-panel { margin-top: 22px; padding: 16px; border: 1px solid var(--border-default); border-radius: var(--radius-lg); background: var(--bg-card); }
.rag-heading { margin-bottom: 14px; }
.field-label,.list-title { display:block; font-weight:600; margin:14px 0 8px; color:var(--text-primary); }
.mode-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:8px; }
.mode-card { padding:10px; border:1px solid var(--border-default); border-radius:8px; background:var(--bg-elevated); color:var(--text-primary); text-align:left; cursor:pointer; }
.mode-card.active { border-color:var(--color-primary); box-shadow:0 0 0 1px var(--color-primary); }
.mode-card small,.knowledge-row small,.qa-row small { display:block; margin-top:4px; color:var(--text-secondary); }
.inherit-hint,.empty-note { color:var(--text-secondary); font-size:12px; }
.advanced-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; margin-top:10px; }
.advanced-grid label { display:flex; flex-direction:column; gap:5px; font-size:12px; }
.toolbar,.search-line,.row-actions { display:flex; gap:8px; align-items:center; margin-top:12px; }
.toolbar.compact { margin-top:8px; }
.primary-button,.secondary-button,.danger-button,.text-button { border:0; border-radius:7px; padding:8px 12px; cursor:pointer; }
.primary-button { background:var(--color-primary); color:white; }
.secondary-button { background:var(--bg-elevated); color:var(--text-primary); border:1px solid var(--border-default); }
.danger-button { background:transparent; color:#ef4444; padding:5px 8px; }
.text-button { background:transparent; color:var(--color-primary); padding:5px 0; }
button:disabled { opacity:.5; cursor:not-allowed; }
.rag-status { margin-top:10px; padding:8px 10px; border-radius:7px; font-size:12px; background:var(--bg-elevated); }
.rag-status.error,.warning-text { color:#ef4444; }.rag-status.success { color:#22c55e; }.rag-status.warning { color:#f59e0b; }
.knowledge-row,.qa-row { display:flex; justify-content:space-between; gap:12px; padding:10px 0; border-bottom:1px solid var(--border-default); }
.qa-editor,.search-test,.knowledge-list { margin-top:16px; }
.qa-editor textarea { margin-top:8px; resize:vertical; }
.qa-row p { margin:5px 0; color:var(--text-secondary); white-space:pre-wrap; }
.search-line .form-input { flex:1; }
.result-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:8px; margin-top:10px; }
.result-group { min-width:0; padding:8px; border:1px solid var(--border-default); border-radius:8px; }
.result-hit { margin-top:7px; padding-top:7px; border-top:1px solid var(--border-default); }
.result-hit b,.result-hit span { display:block; font-size:12px; }
.result-hit span { color:var(--text-secondary); max-height:70px; overflow:hidden; white-space:pre-wrap; }
@media (max-width:760px) { .mode-grid,.result-grid,.advanced-grid { grid-template-columns:1fr; } }
</style>
