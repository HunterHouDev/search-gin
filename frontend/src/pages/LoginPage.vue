<template>
  <q-layout view="lHh Lpr lFf">
    <q-page-container>
      <q-page class="flex flex-center">
        <q-card class="q-pa-md login-card" style="width: 400px">
          <q-card-section>
            <div class="text-h5 text-center q-mb-sm">欢迎使用</div>
            <div class="text-subtitle1 text-center text-grey-7">搜索系统</div>
          </q-card-section>
          
          <q-card-section>
            <q-form @submit="login">
              <q-input
                v-model="username"
                label="用户名"
                required
                :disable="loading"
                :rules="[val => !!val || '用户名不能为空']"
                class="q-mb-md"
              >
                <template v-slot:prepend>
                  <q-icon name="person" />
                </template>
              </q-input>

              <q-input
                v-model="password"
                label="密码"
                type="password"
                required
                :disable="loading"
                :rules="[val => !!val || '密码不能为空']"
              >
                <template v-slot:prepend>
                  <q-icon name="lock" />
                </template>
              </q-input>

              <q-btn
                type="submit"
                color="primary"
                label="登录"
                class="q-mt-lg full-width"
                size="lg"
                :loading="loading"
                :disable="loading"
              >
                <template v-slot:loading>
                  <q-spinner-dots />
                </template>
              </q-btn>
            </q-form>
          </q-card-section>
          
          <q-card-section v-if="errorMsg" class="text-center">
            <q-banner inline-actions class="text-white bg-red-4">
              {{ errorMsg }}
            </q-banner>
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
import { useQuasar } from 'quasar';

const router = useRouter();
const systemProperty = useSystemProperty();
const username = ref('');
const password = ref('');
const loading = ref(false);
const errorMsg = ref('');
const $q = useQuasar();

const login = async () => {
  if (!username.value || !password.value) {
    errorMsg.value = '请输入用户名和密码';
    return;
  }
  
  loading.value = true;
  errorMsg.value = '';
  
  try {
    const response = await fetch('/api/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        username: username.value,
        password: password.value 
      }),
    });
    
    const result = await response.json();
    
    if (result.Code === 200) {
      // 验证通过，将用户信息存储到localStorage中
      systemProperty.expireTime = new Date().getTime() + 1000 * 60 * 60 * 2;
      localStorage.setItem('isAuthenticated', 'true');
      localStorage.setItem('authToken', result.Data.token);
      localStorage.setItem('userRole', result.Data.role);
      localStorage.setItem('username', result.Data.username);
      
      $q.notify({
        type: 'positive',
        message: '登录成功',
        position: 'top',
      });
      
      router.push('/');
    } else {
      errorMsg.value = result.Message || result.message || '用户名或密码错误';
    }
  } catch (error) {
    errorMsg.value = '登录失败，请稍后重试';
    console.error('登录错误:', error);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-card {
  border-radius: 12px;
  box-shadow: 0 4px 28px rgba(0, 0, 0, 0.1);
}

.q-page {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  min-height: 100vh;
}
</style>
