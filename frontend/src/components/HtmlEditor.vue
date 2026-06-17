<template>
  <q-editor
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    :toolbar="toolbar"
    class="html-editor"
  />
</template>

<script setup>
import { useQuasar } from 'quasar';

defineProps({
  modelValue: {
    type: String,
    default: '',
  },
});

defineEmits(['update:modelValue']);

const $q = useQuasar();

const toolbar = [
  [
    {
      label: $q.lang.editor.align,
      icon: $q.iconSet.editor.align,
      fixedLabel: true,
      list: 'only-icons',
      options: ['left', 'center', 'right', 'justify'],
    },
  ],
  ['bold', 'italic', 'strike', 'underline'],
  ['token', 'hr', 'link'],
  ['print', 'fullscreen'],
  [
    {
      label: $q.lang.editor.formatting,
      icon: $q.iconSet.editor.formatting,
      list: 'no-icons',
      options: ['p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'code'],
    },
    'removeFormat',
  ],
  ['quote', 'unordered', 'ordered', 'outdent', 'indent'],
  ['undo', 'redo'],
  ['viewsource'],
];
</script>

<style lang="scss" scoped>
.html-editor {
  min-height: 200px;
  background: var(--q-bg-input);
  border: 1px solid var(--q-border);
  border-radius: 6px;

  :deep(.q-editor__toolbar) {
    background: var(--q-bg-card);
    border-bottom: 1px solid var(--q-border);
  }

  :deep(.q-editor__content) {
    color: var(--q-text-primary);
  }
}
</style>
