<template>
  <div class="skeleton-container">
    <!-- 搜索结果骨架屏 -->
    <div v-if="type === 'search'" class="skeleton-search">
      <div v-for="i in count" :key="i" class="skeleton-item">
        <div class="skeleton-image q-mr-md"></div>
        <div class="skeleton-content">
          <div class="skeleton-title"></div>
          <div class="skeleton-subtitle"></div>
          <div class="skeleton-tags">
            <div class="skeleton-tag"></div>
            <div class="skeleton-tag"></div>
            <div class="skeleton-tag"></div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 卡片骨架屏 -->
    <div v-else-if="type === 'card'" class="skeleton-cards">
      <div v-for="i in count" :key="i" class="skeleton-card">
        <div class="skeleton-card-image"></div>
        <div class="skeleton-card-content">
          <div class="skeleton-title"></div>
          <div class="skeleton-subtitle"></div>
        </div>
      </div>
    </div>
    
    <!-- 列表骨架屏 -->
    <div v-else-if="type === 'list'" class="skeleton-list">
      <div v-for="i in count" :key="i" class="skeleton-list-item">
        <div class="skeleton-list-icon"></div>
        <div class="skeleton-list-content">
          <div class="skeleton-title"></div>
          <div class="skeleton-subtitle"></div>
        </div>
      </div>
    </div>
    
    <!-- 默认骨架屏 -->
    <div v-else class="skeleton-default">
      <div v-for="i in count" :key="i" class="skeleton-line"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps({
  type: {
    type: String,
    default: 'default',
    validator: (v: string) => ['default', 'search', 'card', 'list'].includes(v)
  },
  count: {
    type: Number,
    default: 6
  }
});
</script>

<style lang="scss" scoped>
@keyframes skeleton-pulse {
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
  100% {
    opacity: 1;
  }
}

.skeleton-container {
  width: 100%;
}

// 搜索结果骨架屏
.skeleton-search {
  .skeleton-item {
    display: flex;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 8px;
    background: var(--q-bg-card);
    animation: skeleton-pulse 1.5s ease-in-out infinite;
    
    &:nth-child(odd) {
      animation-delay: 0.1s;
    }
    
    &:nth-child(3n) {
      animation-delay: 0.2s;
    }
  }
  
  .skeleton-image {
    width: 120px;
    height: 80px;
    border-radius: 8px;
    background: var(--q-bg-input);
  }
  
  .skeleton-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  
  .skeleton-title {
    width: 60%;
    height: 16px;
    border-radius: 4px;
    background: var(--q-bg-input);
    margin-bottom: 8px;
  }
  
  .skeleton-subtitle {
    width: 40%;
    height: 12px;
    border-radius: 4px;
    background: var(--q-bg-input);
    margin-bottom: 8px;
  }
  
  .skeleton-tags {
    display: flex;
    gap: 8px;
  }
  
  .skeleton-tag {
    width: 60px;
    height: 20px;
    border-radius: 12px;
    background: var(--q-bg-input);
  }
}

// 卡片骨架屏
.skeleton-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  
  .skeleton-card {
    background: var(--q-bg-card);
    border-radius: 12px;
    overflow: hidden;
    animation: skeleton-pulse 1.5s ease-in-out infinite;
    
    &:nth-child(odd) {
      animation-delay: 0.15s;
    }
    
    &:nth-child(3n) {
      animation-delay: 0.3s;
    }
  }
  
  .skeleton-card-image {
    width: 100%;
    height: 140px;
    background: var(--q-bg-input);
  }
  
  .skeleton-card-content {
    padding: 12px;
  }
  
  .skeleton-title {
    width: 80%;
    height: 14px;
    border-radius: 4px;
    background: var(--q-bg-input);
    margin-bottom: 8px;
  }
  
  .skeleton-subtitle {
    width: 50%;
    height: 10px;
    border-radius: 4px;
    background: var(--q-bg-input);
  }
}

// 列表骨架屏
.skeleton-list {
  .skeleton-list-item {
    display: flex;
    align-items: center;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 4px;
    background: var(--q-bg-card);
    animation: skeleton-pulse 1.5s ease-in-out infinite;
    
    &:nth-child(odd) {
      animation-delay: 0.1s;
    }
    
    &:nth-child(3n) {
      animation-delay: 0.2s;
    }
  }
  
  .skeleton-list-icon {
    width: 40px;
    height: 40px;
    border-radius: 8px;
    background: var(--q-bg-input);
    margin-right: 12px;
  }
  
  .skeleton-list-content {
    flex: 1;
  }
  
  .skeleton-title {
    width: 50%;
    height: 14px;
    border-radius: 4px;
    background: var(--q-bg-input);
    margin-bottom: 6px;
  }
  
  .skeleton-subtitle {
    width: 30%;
    height: 10px;
    border-radius: 4px;
    background: var(--q-bg-input);
  }
}

// 默认骨架屏
.skeleton-default {
  .skeleton-line {
    width: 100%;
    height: 20px;
    border-radius: 4px;
    background: var(--q-bg-input);
    margin-bottom: 12px;
    animation: skeleton-pulse 1.5s ease-in-out infinite;
    
    &:nth-child(odd) {
      animation-delay: 0.1s;
    }
    
    &:nth-child(3n) {
      animation-delay: 0.2s;
    }
    
    &:nth-child(2) {
      width: 90%;
    }
    
    &:nth-child(3) {
      width: 75%;
    }
  }
}
</style>
