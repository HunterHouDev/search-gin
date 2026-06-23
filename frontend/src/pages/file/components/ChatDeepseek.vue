<template>
  <q-dialog ref="dialogRef" @hide="closeWin" @close="closeWin">
    <q-card
      class="q-dialog-plugin"
      style="max-width: 1200px; width: 80vw; min-height: 660px"
    >
      <q-tabs ripple v-model="tab" align="justify" narrow-indicator>
        <q-tab name="普通对话" label="normal" />
      </q-tabs>
      <q-tab-panels v-model="tab" animated style="height: 100%; overflow: auto">
        <q-tab-panel name="normal">
          <q-input
            color="red-12"
            label="发起提问"
            autogrow
            v-model="view.question"
            :dense="false"
            clearable
          >
            <template v-slot:append>
              <q-icon name="ti-search" class="cursor-pointer" @click="askAi" />
            </template>
          </q-input>
          <div class="q-pa-md row justify-between">
            <div style="width: 100%; max-height: 400px">
              <q-chat-message
                v-for="chat in view.messages"
                :name="chat.role"
                :key="chat.content"
                :text="[chat.content]"
                :sent="chat.role == 'user'"
              />
            </div>
          </div>
        </q-tab-panel>
      </q-tab-panels>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useDialogPluginComponent } from 'quasar';

import axios from 'axios';

const tab = ref('normal');

const view = reactive({
  question: '',
  messages: [
    {
      content: 'Hi',
      role: 'user',
    },
    {
      content: 'You are a helpful assistant',
      role: 'system',
    },
  ],
});

const askAi = () => {
  if (view.question && view.question.length > 0) {
    view.messages.unshift({ content: view.question, role: 'user' });
    chatAi(Array.from(view.messages));
  }
};

const chatAi = (msgs: { content: string; role: string }[]) => {
  axios
    .post('/api/chat/deepseek', {
      messages: msgs,
      model: 'deepseek-chat',
    })
    .then((response) => {
      const content = response.data?.Data;
      if (content) {
        view.messages.unshift({ content, role: 'assistant' });
      }
    })
    .catch((error) => {
      console.error('AI 对话失败:', error);
    });
};

const { dialogRef, onDialogCancel } = useDialogPluginComponent();

const open = () => {
  if (dialogRef.value) {
    dialogRef.value.show();
  }
};

const closeWin = () => {
  onDialogCancel();
};

defineExpose({
  open,
});
</script>

<style scoped></style>
