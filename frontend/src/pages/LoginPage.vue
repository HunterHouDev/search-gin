<template>
  <q-layout view="lHh Lpr lFf">
    <q-page-container>
      <q-page class="flex flex-center">
        <q-card class="q-pa-md" style="width: 350px">
          <q-card-section>
            <div class="text-h6 text-center">登录</div>
          </q-card-section>
          <q-card-section>
            <q-form @submit="login">
              <q-input
                v-model="password"
                label="密码"
                type="password"
                required
                debounce="500"
                @update:model-value="login"
              />
              <q-btn
                type="submit"
                color="primary"
                label="登录"
                class="q-mt-md full-width"
              />
            </q-form>
          </q-card-section>
        </q-card>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { useSystemProperty } from 'src/stores/System';
import { ref } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();
const systemProperty = useSystemProperty();
const password = ref('');

const login = () => {
  // 这里可以添加实际的登录验证逻辑
  // 例如，调用API验证用户名和密码是否匹配
  // 如果验证通过，将用户信息存储到localStorage中
  // 然后重定向到首页
  // 这里只是一个简单的示例，实际应用中需要根据具体情况进行修改
  if (password.value !== 'qwer') {
    return;
  }
  // 验证通过，将用户信息存储到localStorage中
  systemProperty.expireTime = new Date().getTime() + 1000 * 60 * 60 * 2;
  localStorage.setItem('isAuthenticated', 'true');
  router.push('/');
};
</script>
