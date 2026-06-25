<template>
  <div class="empty-state">
    <!-- SVG 插画 -->
    <div class="empty-illustration">
      <svg v-if="icon === 'search'" width="96" height="96" viewBox="0 0 96 96" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="40" cy="40" r="24" stroke="currentColor" stroke-width="3" opacity="0.3"/>
        <path d="M58 58L76 76" stroke="currentColor" stroke-width="3" stroke-linecap="round" opacity="0.3"/>
        <line x1="28" y1="28" x2="36" y2="36" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="44" y1="28" x2="36" y2="36" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
      </svg>

      <svg v-else-if="icon === 'file'" width="96" height="96" viewBox="0 0 96 96" fill="none" xmlns="http://www.w3.org/2000/svg">
        <rect x="24" y="16" width="48" height="64" rx="4" stroke="currentColor" stroke-width="3" opacity="0.3"/>
        <line x1="36" y1="40" x2="60" y2="40" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="36" y1="50" x2="54" y2="50" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="36" y1="60" x2="48" y2="60" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
      </svg>

      <svg v-else-if="icon === 'network'" width="96" height="96" viewBox="0 0 96 96" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="48" cy="48" r="32" stroke="currentColor" stroke-width="3" opacity="0.3"/>
        <circle cx="48" cy="48" r="16" stroke="currentColor" stroke-width="2" opacity="0.15"/>
        <line x1="48" y1="16" x2="48" y2="34" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="48" y1="62" x2="48" y2="80" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="16" y1="48" x2="34" y2="48" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
        <line x1="62" y1="48" x2="80" y2="48" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.15"/>
      </svg>

      <svg v-else width="96" height="96" viewBox="0 0 96 96" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="48" cy="48" r="32" stroke="currentColor" stroke-width="3" opacity="0.3"/>
        <path d="M36 36L60 60" stroke="currentColor" stroke-width="3" stroke-linecap="round" opacity="0.15"/>
        <path d="M60 36L36 60" stroke="currentColor" stroke-width="3" stroke-linecap="round" opacity="0.15"/>
      </svg>
    </div>

    <!-- 标题 -->
    <h4 class="empty-title">{{ title }}</h4>

    <!-- 描述 -->
    <p v-if="description" class="empty-description">{{ description }}</p>

    <!-- 操作按钮 -->
    <q-btn
      v-if="actionLabel"
      :label="actionLabel"
      :color="actionColor"
      :icon="actionIcon"
      unelevated
      no-caps
      class="q-mt-md"
      @click="$emit('action')"
    />
  </div>
</template>

<script setup lang="ts">
defineProps({
  /** 插画类型: search | file | network | default */
  icon: {
    type: String,
    default: 'default',
    validator: (v: string) => ['search', 'file', 'network', 'default'].includes(v),
  },
  /** 主标题 */
  title: {
    type: String,
    required: true,
  },
  /** 详细描述 */
  description: {
    type: String,
    default: '',
  },
  /** 操作按钮文案 — 传此值时显示按钮 */
  actionLabel: {
    type: String,
    default: '',
  },
  /** 操作按钮图标 */
  actionIcon: {
    type: String,
    default: '',
  },
  /** 操作按钮颜色 */
  actionColor: {
    type: String,
    default: 'primary',
  },
});

defineEmits<{
  action: [];
}>();
</script>

<style lang="scss" scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
  user-select: none;
}

.empty-illustration {
  color: var(--q-text-muted);
  opacity: 0.6;
  margin-bottom: 8px;
}

.empty-title {
  font-family: var(--font-heading);
  font-size: 18px;
  font-weight: 600;
  color: var(--q-text-primary);
  margin: 12px 0 4px;
}

.empty-description {
  font-size: 14px;
  color: var(--q-text-muted);
  line-height: 1.6;
  max-width: 320px;
  margin: 4px 0 0;
}
</style>
