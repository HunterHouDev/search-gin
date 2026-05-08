<template>
  <q-knob
    show-value
    v-model="systemProperty.videoOptions.volume"
    :thickness="0.1"
    :color="props.color"
    :center-color="props.centerColor"
    size="md"
    flat
    :max="1"
    :min="0"
    :step="0.1"
    @update:modelValue="volumeUpdate"
  >
    {{ systemProperty.videoOptions.volume }}
    <!-- <q-icon name="volume_up" size="sm" /> -->
  </q-knob>
</template>
<script setup>
import { useSystemProperty } from 'stores/System';
import { onKeyStroke } from '@vueuse/core';

const systemProperty = useSystemProperty();

const props = defineProps({
  color: { type: String, default: 'red' },
  centerColor: { type: String, default: undefined },
});

const emmits = defineEmits(['volumeUpdate', 'volumeUp']);

const volumeUpdate = (val) => {
  if (!isNaN(val)) {
    systemProperty.videoOptions.volume = val;
    emmits('volumeUpdate', val);
  }
};
const volumeUp = (val) => {
  if (!isNaN(val)) {
    if (
      systemProperty.videoOptions.volume + val <= 1 &&
      systemProperty.videoOptions.volume + val >= 0
    ) {
      systemProperty.videoOptions.volume = Number(
        (systemProperty.videoOptions.volume + val).toFixed(1)
      );
    }
    emmits('volumeUp', val);
  }
};

onKeyStroke(['ArrowUp'], () => {
  if (systemProperty.PlayingMovie.Id) {
    volumeUp(0.1);
  }
});
onKeyStroke(['ArrowDown'], () => {
  if (systemProperty.PlayingMovie.Id) {
    volumeUp(-0.1);
  }
});
</script>
