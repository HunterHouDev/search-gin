<template>
  <div
    class="justify-start q-gutter-sm"
    style="
      width: 32rem;
      padding: 1rem 1rem;
      border-radius: 10px;
      background-color: rgba(250, 250, 250, 0.9);
    "
  >
    

    <div class="row w100">
      <div class="col"><q-btn flat dense> 播放器亮度 </q-btn></div>
      <q-btn flat dense>
        <q-slider
          v-model:model-value="systemProperty.videoOptions.brightness"
          color="blue"
          trackColor="grey"
          dense
          :min="20"
          :max="100"
          :markers="100"
          style="margin: 2px 30px; width: 300px; height: 10px"
        >
        </q-slider>
        {{ systemProperty.videoOptions.brightness }}%
      </q-btn>
    </div>
    <div class="row w100">
      <div class="col">
        <q-btn flat dense> 播放器种草后 </q-btn>
        <q-radio
          v-model="systemProperty.addPlayingTagGoNext"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="true"
          label="下一个"
        />
        <q-radio
          v-model="systemProperty.addPlayingTagGoNext"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="false"
          label="上一个"
        />
      </div>
    </div>
    <div class="row w100">
      <div class="col">
        <q-btn flat dense> 记忆播放位置 </q-btn>
        <q-radio
          v-model="systemProperty.playerReLocation"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="true"
          label="记忆"
        />
        <q-radio
          v-model="systemProperty.playerReLocation"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="false"
          label="忘记"
        />
      </div>
    </div>

    <q-btn-toggle
      spread
      no-caps
      glossy
      v-model="systemProperty.videoOptions.playerMode"
      :options="[
        {
          label: '缩放',
          value: 'contain',
        },
        {
          label: '拉伸',
          value: 'fill',
        },
        {
          label: '原始',
          value: 'none',
        },
        {
          label: '剪切',
          value: 'cover',
        },
      ]"
      toggle-color="red"
    />
    <q-btn-toggle
      v-if="props.widescreen"
      spread
      no-caps
      glossy
      v-model="systemProperty.videoOptions.widescreen"
      :options="[
        {
          label: '适应横屏',
          value: true,
        },
        {
          label: '适应竖屏',
          value: false,
        },
      ]"
      toggle-color="red"
    />
    <q-btn-toggle
          spread
          no-caps
          glossy
          v-model="systemProperty.videoOptions.rotate"
          :options="[
            {
              label: '正常',
              value: 'rotate(0deg)',
            },
            {
              label: '转90',
              value: 'rotate(90deg)',
            },
            {
              label: '上下',
              value: 'rotate(180deg)',
            },
            {
              label: '左右',
              value: 'scaleX(-1)',
            },
            // {
            //   label: '放大',
            //   value: 'scale(.5)',
            // },
          ]"
          toggle-color="red"
        />
  </div>
</template>

<script setup>
import { useSystemProperty } from 'src/stores/System';

const systemProperty = useSystemProperty();

const props = defineProps({
  widescreen: {
    type: Boolean,
    default: false,
  },
});
</script>
