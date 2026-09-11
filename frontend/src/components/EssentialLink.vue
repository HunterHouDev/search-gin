<template>
  <q-item
    clickable
    tag="a"
    :href="href"
    style="padding: 0"
    @pointerenter="refresh"
    @click="navigate"
  >
    <q-btn
      style="margin: 1px 8px;scale:1.2"
      flat
      dense
      v-if="props.icon"
      :icon="icon"
      :color="currentPath == props.link ? 'red' : 'white'"
      >{{ props.title }}</q-btn
    >
  </q-item>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { useSessionLink } from 'src/composables/useSessionLink';

const props = defineProps({
  link: {
    type: String,
    required: true
  },
  icon: {
    type: String,
    default: null
  },
  title: {
    type: String,
    required: true
  }
});

const route = useRoute();

const currentPath = computed(() => route.path);

// 菜单项是静态路由配置：href 挂登录态快照，浏览器原生"在新窗口中打开"才不会丢登录态
const { href, refresh, navigate } = useSessionLink(props.link);
</script>
