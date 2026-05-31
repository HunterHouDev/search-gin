<template>
  <div
    ref="target"
    style="width: 100%; height: 100%; padding: 1px"
    @blur="editStyle == false"
  >
    <span v-if="!editStyle">
      <q-chip
        dense
        color="orange"
        text-color="white"
        v-for="(item, idx) in value"
        :key="item"
      >
        <q-btn
          dense
          color="red"
          size="sm"
          icon="ti-arrow-left"
          v-if="idx != 0"
          flat
          @click="moveRight(item, -1)"
          @contextmenu="
            (e) => {
              moveRight(item, -5);
              e.returnValue = false;
            }
          "
        />
        {{ item }}
        <q-btn
          dense
          flat
          color="red"
          size="sm"
          icon="ti-arrow-right"
          v-if="idx != value.length - 1"
          @click="moveRight(item, 1)"
          @contextmenu="
            (e) => {
              moveRight(item, 5);
              e.returnValue = false;
            }
          "
        />
      </q-chip>
      <q-btn
        dense
        flat
        color="red"
        icon="add"
        v-if="editStyle == false"
        @click="editStyle = true"
      />
    </span>
    <div v-if="editStyle">
      <q-checkbox
        class="checkItem"
        v-model="value"
        v-for="item in props.options"
        :key="item"
        :val="item"
        :label="item"
        color="teal"
        @update:model-value="updateValue"
      />
      <q-btn
        flat
        color="red"
        icon="ti-close"
        v-if="editStyle == true"
        @click="editStyle = false"
      />
    </div>
  </div>
</template>
<script setup>
import { onMounted, ref, watch } from 'vue';

const emits = defineEmits(['update:model-value', 'onchange']);

const target = ref(null);

const value = ref([]);
const editStyle = ref(false);
const props = defineProps({
  modelValue: {
    type: Array,
    default: () => [],
  },
  options: {
    type: Array,
    default: () => [],
  },
});

const updateValue = (arr) => {
  emits('update:model-value', arr);
  emits('onchange', arr);
};
watch(
  () => props.modelValue,
  (e) => {
    value.value = e;
  }
);

onMounted(() => {
  const arr = [];
  if (!props.options) return;
  props.options.forEach((item) => {
    if (props.modelValue && props.modelValue.includes(item)) {
      arr.push(item);
    }
  });
  value.value = arr;
  emits('update:model-value', arr);
  emits('onchange', arr);
});

const moveRight = (str, step) => {
  console.log(str, step);
  if (!value.value) {
    value.value = [];
  }
  if (value.value.indexOf(str) < 0) {
    return;
  }
  const idx = value.value.indexOf(str);
  const newIdx = idx + step;
  if (newIdx < 0 || newIdx >= value.value.length) {
    return;
  }
  // 交换元素位置
  const item = value.value.splice(idx, 1)[0];
  value.value.splice(newIdx, 0, item);
  // 触发更新
  emits('update:model-value', value.value);
  emits('onchange', value.value);
};
</script>
<style lang="scss">
.checkItem {
  width: 8rem;
}
</style>
