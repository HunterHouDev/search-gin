<template>
  <q-layout view="lHh Lpr lFf" :class="{ 'theme-natural': systemProperty.theme === 'natural' }">
    <q-page-container>
      <q-page class="login-page flex flex-center">
        <div class="login-bg-glow"></div>

        <q-card class="login-card q-pa-lg" flat>
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
                初始化设置
              </div>
              <div class="text-caption text-muted q-mt-xs">
                首次使用请设置管理员密码
              </div>
            </div>
          </q-card-section>

          <q-card-section class="q-pt-lg">
            <q-form @submit="handleSetup">
              <q-input
                v-model="password"
                type="password"
                label="管理员密码"
                outlined
                dense
                class="q-mb-md"
                :rules="[val => !!val || '密码不能为空']"
              />
              <q-input
                v-model="confirmPassword"
                type="password"
                label="确认密码"
                outlined
                dense
                class="q-mb-lg"
                :rules="[val => val === password || '两次密码不一致']"
              />

              <q-btn
                type="submit"
                color="primary"
                class="full-width"
                label="初始化"
                :loading="loading"
              />

              <q-btn
                flat
                color="grey-6"
                class="full-width q-mt-sm"
                label="跳转登录页"
                @click="router.push('/login')"
              />
            </q-form>
          </q-card-section>
        </q-card>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useQuasar } from 'quasar';
import { commonAxios } from 'src/boot/axios';
import { useSystemProperty } from 'src/stores/System';

const $q = useQuasar();
const router = useRouter();
const systemProperty = useSystemProperty();

const password = ref('');
const confirmPassword = ref('');
const loading = ref(false);

const handleSetup = async () => {
  if (!password.value) {
    $q.notify({ type: 'negative', message: '密码不能为空' });
    return;
  }
  if (password.value !== confirmPassword.value) {
    $q.notify({ type: 'negative', message: '两次密码不一致' });
    return;
  }

  loading.value = true;
  try {
    const res = await commonAxios().post('/api/init/setup', {
      password: password.value,
    });
    if (res.data.Code === 200) {
      $q.notify({ type: 'positive', message: '初始化成功，请登录' });
      router.push('/login');
    } else {
      $q.notify({ type: 'negative', message: res.data.Message || '初始化失败' });
    }
  } catch (e) {
    $q.notify({ type: 'negative', message: '初始化失败: ' + (e.response?.data?.Message || e.message) });
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
}

/* 背景光晕 */
.login-bg-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 500px;
  height: 500px;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, transparent 70%);
  pointer-events: none;
  animation: glowPulse 4s ease-in-out infinite;
}

@keyframes glowPulse {
  0%, 100% { opacity: 0.5; transform: translate(-50%, -50%) scale(1); }
  50% { opacity: 1; transform: translate(-50%, -50%) scale(1.15); }
}

/* 登录卡片 */
.login-card {
  width: 400px;
  max-width: 90vw;
  background: rgba(255, 255, 255, 0.06) !important;
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.3);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.login-card:hover {
  box-shadow: 0 12px 48px rgba(0, 0, 0, 0.4);
}

/* Logo */
.login-logo {
  display: inline-flex;
  margin-bottom: 8px;
}

.login-logo svg {
  filter: drop-shadow(0 2px 8px rgba(99, 102, 241, 0.4));
}

.logo-bg {
  color: #6366f1;
}

/* 品牌区域 */
.login-brand {
  padding: 12px 0;
}

.login-brand .text-h5 {
  color: #fff;
  letter-spacing: 0.5px;
}

.login-brand :deep(.text-muted) {
  color: rgba(255, 255, 255, 0.5) !important;
}

/* 输入框适配暗色背景 */
.login-card :deep(.q-field__label) {
  color: rgba(255, 255, 255, 0.6) !important;
}

.login-card :deep(.q-field__control) {
  background: rgba(255, 255, 255, 0.05) !important;
  border-radius: 8px;
}

.login-card :deep(.q-field__control::before) {
  border-color: rgba(255, 255, 255, 0.12) !important;
}

.login-card :deep(.q-field__native) {
  color: #fff !important;
}

.login-card :deep(.q-field__control:hover::before) {
  border-color: rgba(99, 102, 241, 0.4) !important;
}

.login-card :deep(.q-field--highlighted .q-field__control::after) {
  background: #6366f1 !important;
}

/* 按钮样式 */
.login-card :deep(.q-btn) {
  border-radius: 8px;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: none;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%) !important;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.login-card :deep(.q-btn:hover) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.login-card :deep(.q-btn:active) {
  transform: translateY(0);
}

/* Natural 主题覆写 */
.theme-natural .login-page {
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
}

.theme-natural .login-bg-glow {
  background: radial-gradient(circle, rgba(52, 211, 153, 0.12) 0%, transparent 70%);
}

.theme-natural .login-card {
  background: rgba(255, 255, 255, 0.05) !important;
  border-color: rgba(52, 211, 153, 0.15);
}

.theme-natural .login-logo svg {
  filter: drop-shadow(0 2px 8px rgba(52, 211, 153, 0.3));
}

.theme-natural .logo-bg {
  color: #34d399;
}

.theme-natural .login-card :deep(.q-field--highlighted .q-field__control::after) {
  background: #34d399 !important;
}

.theme-natural .login-card :deep(.q-btn) {
  background: linear-gradient(135deg, #34d399 0%, #10b981 100%) !important;
}

/* 响应式 */
@media (max-width: 480px) {
  .login-card {
    padding: 20px !important;
  }

  .login-card .q-card__section:first-child {
    padding-top: 12px !important;
  }

  .login-card .q-card__section:last-child {
    padding-bottom: 12px !important;
  }

  .login-bg-glow {
    width: 300px;
    height: 300px;
  }
}
</style>
