<template>
  <div
    style="
      padding: 4px;
      z-index: 999;
      position: fixed;
      margin-left: 8rem;
      width: 8rem;
      display: flex;
      justify-content: space-between;
      background-color: rgba(0, 0, 0, 0.8);
      border-radius: 10px;
    "
    v-if="view.showDelete"
  >
    <q-btn
      dense
      icon="ti-trash"
      color="red"
      size="md"
      @mouseover="deleteMouseIn"
      @mouseout="deleteMouseOut"
      @click="
        view.showDelete = false;
        picDelete(-1);
      "
      ><q-icon name="ti-control-skip-backward" />
      <q-tooltip class="bg-white text-primary">删除并播放上一个</q-tooltip>
    </q-btn>
    <q-btn
      dense
       size="md"
      icon-right="ti-trash"
      @mouseover="deleteMouseIn"
      @mouseout="deleteMouseOut"
      color="red"
      @click="
        view.showDelete = false;
        picDelete(1);
      "
      ><q-icon name="ti-control-skip-forward" /><q-tooltip
        class="bg-white text-primary"
        >删除并播放下一个</q-tooltip
      ></q-btn
    >
  </div>

  <q-btn
    v-if="!view.showDelete"
    icon="ti-trash"
    color="red"
    size="md"
    align="center"
    :dense="props.dense"
    :flat="props.flat"
    @click="
      () => {
        deleteMouseIn();
        deleteMouseOut();
      }
    "
    ><q-tooltip class="bg-white text-primary">选项删除</q-tooltip>
  </q-btn>
</template>
<script setup>
import { inject, onUnmounted, reactive } from 'vue';
import { useQuasar } from 'quasar';
import { DeleteFile } from 'components/api/searchAPI';
const $q = useQuasar();
const props = defineProps({
  currentData: {
    type: Object,
    default: () => ({}),
  },
  dense: {
    type: Boolean,
    default: false,
  },
  flat: {
    type: Boolean,
    default: false,
  },
});

const view = reactive({
  showDelete: false,
});

const emmits = defineEmits(['nextOne', 'prevOne']);

let deleteTimeout = null;
const deleteMouseIn = () => {
  clearTimeout(deleteTimeout);
  view.showDelete = true;
};
const deleteMouseOut = () => {
  deleteTimeout = setTimeout(() => {
    view.showDelete = false;
  }, 3000);
};

const fetchToUpdateList = inject('fetchToUpdateList', () => undefined);

const picDelete = async (n) => {
  const idToDelete = props.currentData.Id;
  if (n && n > 0) {
    emmits('nextOne');
  } else {
    emmits('prevOne');
  }
  setTimeout(async () => {
    const { Code, Message } = await DeleteFile(idToDelete);
    if (Code !== 200) {
      $q.notify({ message: `${Message}`, position: 'bottom-left' });
    }
    fetchToUpdateList(props.currentData);
  }, 3000);
};

onUnmounted(() => {
  clearTimeout(deleteTimeout);
});
</script>
