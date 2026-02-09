<template>
  <div class="app-shell" :class="{ 'auth-mode': isAuthRoute }">
    <TopBar v-if="!isAuthRoute" />
    <main class="content" v-if="!isAuthRoute">
      <router-view />
    </main>
    <router-view v-else />
    <TaskDrawer v-if="!isAuthRoute" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import TopBar from './components/TopBar.vue'
import TaskDrawer from './components/TaskDrawer.vue'

const route = useRoute()
const isAuthRoute = computed(() => route.path === '/login')
</script>
