<template>
  <q-item clickable tag="a" target="_self" :to="props.link" style="padding: 0">
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
</script>
