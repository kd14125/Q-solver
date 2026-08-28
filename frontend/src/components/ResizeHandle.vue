<template>
  <div class="resize-overlay" aria-label="拖动边框调整窗口大小">
    <span v-for="edge in edges" :key="edge" class="resize-zone" :class="`resize-${edge}`"
      @mousedown="startResize($event, edge)" />
  </div>
</template>

<script setup>
import { WindowGetSize, WindowSetSize, WindowGetPosition, WindowSetPosition } from '../../wailsjs/runtime/runtime'
import { UpdateSettings, GetSettings } from '../../wailsjs/go/main/App'

const MIN_WIDTH = 420
const MIN_HEIGHT = 350
const edges = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']
let resizing = false, direction = '', startX = 0, startY = 0, startWidth = 0, startHeight = 0, startLeft = 0, startTop = 0

async function startResize(event, edge) {
  if (resizing) return
  event.preventDefault(); event.stopPropagation()
  const [size, position] = await Promise.all([WindowGetSize(), WindowGetPosition()])
  resizing = true; direction = edge; startX = event.screenX; startY = event.screenY
  startWidth = size.w; startHeight = size.h; startLeft = position.x; startTop = position.y
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize, { once: true })
  document.body.style.cursor = `${edge}-resize`; document.body.style.userSelect = 'none'
}

function onResize(event) {
  if (!resizing) return
  const dx = event.screenX - startX, dy = event.screenY - startY
  const fromLeft = direction.includes('w'), fromTop = direction.includes('n')
  const width = Math.max(MIN_WIDTH, startWidth + (fromLeft ? -dx : direction.includes('e') ? dx : 0))
  const height = Math.max(MIN_HEIGHT, startHeight + (fromTop ? -dy : direction.includes('s') ? dy : 0))
  const left = fromLeft ? startLeft + startWidth - width : startLeft
  const top = fromTop ? startTop + startHeight - height : startTop
  if (fromLeft || fromTop) WindowSetPosition(left, top)
  WindowSetSize(width, height)
}

async function stopResize() {
  if (!resizing) return
  resizing = false; document.removeEventListener('mousemove', onResize)
  document.body.style.cursor = ''; document.body.style.userSelect = ''
  try {
    const size = await WindowGetSize(), settings = await GetSettings()
    settings.windowWidth = size.w; settings.windowHeight = size.h
    await UpdateSettings(JSON.stringify(settings))
  } catch (error) { console.error('保存窗口尺寸失败:', error) }
}
</script>

<style scoped>
.resize-overlay { position: fixed; inset: 0; z-index: 99999; pointer-events: none; }
.resize-zone { position: absolute; display: block; pointer-events: auto; }
.resize-n, .resize-s { left: 10px; right: 10px; height: 8px; }
.resize-n { top: 0; cursor: n-resize; } .resize-s { bottom: 0; cursor: s-resize; }
.resize-e, .resize-w { top: 10px; bottom: 10px; width: 8px; }
.resize-e { right: 0; cursor: e-resize; } .resize-w { left: 0; cursor: w-resize; }
.resize-ne, .resize-nw, .resize-se, .resize-sw { width: 16px; height: 16px; }
.resize-ne { top: 0; right: 0; cursor: ne-resize; } .resize-nw { top: 0; left: 0; cursor: nw-resize; }
.resize-se { bottom: 0; right: 0; cursor: se-resize; } .resize-sw { bottom: 0; left: 0; cursor: sw-resize; }
.resize-zone:hover { background: rgba(16, 185, 129, .12); }
</style>
