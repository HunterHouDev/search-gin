<template>
  <q-layout view="lHh Lpr lFf" :class="{ 'theme-natural': systemProperty.theme === 'natural' }">
    <q-page-container>
      <q-page class="login-page flex flex-center">
        <!-- 背景装饰粒子 -->
        <div class="login-bg-glow"></div>

        <q-card class="login-card q-pa-lg" flat>
          <!-- 品牌标识 -->
          <q-card-section class="text-center q-pb-none">
            <div class="login-brand">
              <div class="login-logo">
                <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <rect width="48" height="48" rx="12" fill="currentColor" class="logo-bg"/>
                  <path d="M14 18C14 15.7909 15.7909 14 18 14H30C32.2091 14 34 15.7909 34 18V30C34 32.2091 32.2091 34 30 34H18C15.7909 34 14 32.2091 14 30V18Z" stroke="white" stroke-width="2.5" fill="none"/>
                  <circle cx="24" cy="24" r="4" fill="white"/>
                  <path d="M20 14L24 20L28 14" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="text-h5 q-mt-sm text-weight-bold">
                搜索系统
              </div>
              <div class="text-caption text-muted q-mt-xs">
                本地文件搜索与管理平台
              </div>
            </div>
          </q-card-section>

          <!-- 登录表单 -->
          <q-card-section class="q-pt-lg">
            <q-form @submit="login" class="q-gutter-md">
              <q-input
                v-model="username"
                label="用户名"
                outlined
                :disable="loading"
                class="login-input"
                @update:model-value="onUsernameChange"
              >
                <template v-slot:prepend>
                  <q-icon name="person" size="20px" class="input-icon" />
                </template>
              </q-input>

              <q-input
                v-model="password"
                label="密码"
                outlined
                type="password"
                required
                :disable="loading"
                :rules="[val => !!val || '密码不能为空']"
                class="login-input"
              >
                <template v-slot:prepend>
                  <q-icon name="lock" size="20px" class="input-icon" />
                </template>
              </q-input>

              <q-btn
                type="submit"
                label="登录"
                class="login-btn q-mt-sm"
                size="lg"
                :loading="loading"
                :disable="loading"
                unelevated
                no-caps
                rounded
              >
                <template v-slot:loading>
                  <q-spinner-dots size="24px" />
                </template>
              </q-btn>

              <!-- 错误提示 -->
              <transition name="fade">
                <div v-if="errorMsg" class="login-error">
                  <q-icon name="error_outline" size="18px" class="q-mr-xs" />
                  {{ errorMsg }}
                </div>
              </transition>
            </q-form>
          </q-card-section>

          <!-- 版本信息 -->
          <q-card-section class="text-center q-pb-none q-pt-sm">
            <div class="text-caption text-muted" style="opacity: 0.5;">
              v0.1.0
            </div>
          </q-card-section>
        </q-card>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { useSystemProperty } from 'src/stores/System';
import { usePermissionStore } from 'src/stores/permission';
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useQuasar } from 'quasar';
import { commonAxios } from 'src/boot/axios';

const router = useRouter();
const systemProperty = useSystemProperty();
const username = ref('');
const password = ref('');
const loading = ref(false);
const errorMsg = ref('');
const $q = useQuasar();

// 输入时自动过滤非英文字符（仅保留字母、数字、下划线、连字符）
const onUsernameChange = (val: string | number | null) => {
  username.value = (val ?? '').toString().replace(/[^a-zA-Z0-9_.-]/g, '');
};

// 同步主题到 body（登录页独立于 MainLayout）
onMounted(() => {
  document.body.classList.toggle('theme-natural', systemProperty.theme === 'natural');
});

const login = async () => {
  if (!username.value) {
    errorMsg.value = '请输入用户名';
    return;
  }
  if (!/^[a-zA-Z]/.test(username.value)) {
    errorMsg.value = '用户名仅支持英文字母开头';
    return;
  }
  if (!/^[a-zA-Z][a-zA-Z0-9_.-]*$/.test(username.value)) {
    errorMsg.value = '用户名仅支持英文、数字、下划线和连字符';
    return;
  }
  if (!password.value) {
    errorMsg.value = '请输入密码';
    return;
  }

  loading.value = true;
  errorMsg.value = '';

  try {
    const response = await commonAxios().post('/api/login', {
      username: username.value,
      password: password.value,
    });

    const result = response.data;

    if (result.Code === 200) {
      systemProperty.expireTime = new Date().getTime() + 1000 * 60 * 60 * 2;
      sessionStorage.setItem('isAuthenticated', 'true');
      sessionStorage.setItem('authToken', result.Data.token);
      sessionStorage.setItem('userRole', result.Data.role);
      sessionStorage.setItem('username', result.Data.username);

      const permStore = usePermissionStore();
      permStore.setFromLogin(result.Data.role, result.Data.username, result.Data.permissions || []);

      $q.notify({
        type: 'positive',
        message: '登录成功',
        position: 'top',
      });

      router.push('/');
    } else {
      errorMsg.value = result.Message || result.message || '用户名或密码错误';
    }
  } catch (error: any) {
    errorMsg.value = error?.response?.data?.message || '登录失败，请稍后重试';
  } finally {
    loading.value = false;
  }
};
</script>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
}

// 星空模式背景
body:not(.theme-natural) .login-page {
  background: var(--q-bg-page);
  background-image: var(--q-bg-stars);
}

// 自然模式背景
body.theme-natural .login-page {
  background: var(--q-bg-gradient);
}

// 背景辉光装饰
.login-bg-glow {
  position: absolute;
  width: 600px;
  height: 600px;
  border-radius: 50%;
  pointer-events: none;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);

  body:not(.theme-natural) & {
    background: radial-gradient(circle, rgba(11, 105, 229, 0.08) 0%, transparent 70%);
  }

  body.theme-natural & {
    background: radial-gradient(circle, rgba(30, 58, 95, 0.06) 0%, transparent 70%);
  }
}

// 登录卡片
.login-card {
  width: 400px;
  max-width: 90vw;
  background: var(--q-bg-card) !important;
  border: 1px solid var(--q-border);
  border-radius: 12px;
  backdrop-filter: blur(12px);
  position: relative;
  z-index: 1;
  transition: background 0.4s ease, border-color 0.4s ease;
}

// 品牌标识
.login-brand {
  .login-logo {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    border-radius: 16px;
    margin-bottom: 8px;

    body:not(.theme-natural) & {
      .logo-bg { color: var(--q-primary); }
    }
    body.theme-natural & {
      .logo-bg { color: var(--q-primary); }
    }
  }
}

// 输入框
.login-input {
  :deep(.q-field__control) {
    border-radius: 8px !important;
    background: var(--q-bg-input) !important;
    transition: border-color 0.3s ease, background 0.4s ease;
  }

  :deep(.q-field__native) {
    padding-left: 4px;
  }

  .input-icon {
    color: var(--q-text-muted);
    opacity: 0.7;
  }
}

// 登录按钮
.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.5px;
  transition: transform 0.15s ease, box-shadow 0.2s ease;

  body:not(.theme-natural) & {
    background: linear-gradient(135deg, #0b69e5, #7c3aed) !important;
    color: white !important;
  }
  body.theme-natural & {
    background: var(--q-primary) !important;
    color: white !important;
  }

  &:active {
    transform: scale(0.97);
  }

  &:not(:disabled):hover {
    box-shadow: 0 4px 20px rgba(11, 105, 229, 0.3);
  }
}

// 错误提示
.login-error {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  color: var(--q-danger);

  body:not(.theme-natural) & {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
  }
  body.theme-natural & {
    background: rgba(239, 68, 68, 0.08);
    border: 1px solid rgba(239, 68, 68, 0.15);
  }
}
</style>
