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
    // chatAi(Array.from(view.messages).reverse());
    chatAi(Array.from(view.messages));
  }
};

const chatAi = (msgs: { content: string; role: string; }[]) => {
  let data = JSON.stringify({
    // "messages": [
    //   {
    //     "content": "You are a helpful assistant",
    //     "role": "system"
    //   },
    //   {
    //     "content": "Hi",
    //     "role": "user"
    //   }
    // ],
    messages: msgs,
    model: 'deepseek-chat',
    frequency_penalty: 0,
    max_tokens: 2048,
    presence_penalty: 0,
    response_format: {
      type: 'text',
    },
    stop: null,
    stream: false,
    stream_options: null,
    temperature: 1,
    top_p: 1,
    tools: null,
    tool_choice: 'none',
    logprobs: false,
    top_logprobs: null,
  });

  let config = {
    method: 'post',
    maxBodyLength: Infinity,
    url: 'https://api.deepseek.com/chat/completions',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      Authorization: 'Bearer sk-5e20928ea69d465d88eaa9e02665e6a2',
    },
    data: data,
  };

  axios(config)
    .then((response) => {
      console.log(JSON.stringify(response.data));
      view.messages = response.data.messages
    })
    .catch((error) => {
      console.log(error);
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
  // window.location.reload();
};

defineExpose({
  open,
});
</script>

<style scoped></style>
