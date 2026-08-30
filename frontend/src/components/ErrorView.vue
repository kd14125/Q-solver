<template>
  <div class="error-container">
    <div class="error-title-large">{{ errorState.title }}</div>
    <div class="error-desc">{{ errorState.desc }}</div>
    
    <!-- 极简模式：只保留快捷键提示 -->
    <div class="error-shortcut-hint" style="margin-top: 20px;">
      按下 <span class="shortcut-tag">{{ solveShortcut }}</span> 重试
    </div>
  </div>
</template>

<script setup>
defineProps({
  errorState: Object,
  solveShortcut: String
})
</script>

<style scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: var(--space-5);
  color: var(--ai-text-color);
  opacity: var(--ai-text-opacity);
  background: transparent;
}

.error-title-large {
  font-size: var(--text-xl);
  font-weight: 700;
  color: inherit;
  margin-bottom: var(--space-2);
}

.error-desc {
  font-size: var(--text-sm);
  color: inherit;
  margin-bottom: var(--space-6);
  max-width: 80%;
  line-height: 1.5;
}

.error-shortcut-hint {
  font-size: var(--text-sm);
  color: inherit;
  margin-bottom: var(--space-5);
}

.shortcut-tag {
  background: transparent;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-weight: bold;
  color: inherit;
  border: 0;
}
</style>
