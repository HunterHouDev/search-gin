<template>
  <q-item clickable tag="a" target="_self" :to="props.link" style="padding: 0">
    <q-btn
      style="margin: 1px 8px;scale:1.2"
      flat
      dense
      v-if="props.icon"
      :icon="icon"
      :color="currentPath == link ? 'red' : ''"
      >{{ props.title }}</q-btn
    >
  </q-item>
</template>

<script setup>
// 引入 computed
import { computed } from 'vue';
// 引入 useRoute
import { useRoute } from 'vue-router';

// 问题出在 defineProps 泛型用法上，在 <script setup> 中，defineProps 可以使用类型注解的方式定义 props
// 修正为使用类型注解来定义 props
const props = defineProps({
  // 这里假设 EssentialLinkProps 包含 link、icon 和 title 属性，需要根据实际情况调整
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

const { path } = useRoute();

const currentPath = computed(() => {
  console.log(path);
  return useRoute().path;
});

</script>
