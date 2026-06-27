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
