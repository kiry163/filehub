<template>
  <div class="app-shell">
    <TopBar />
    <div class="shell">
      <Sidebar />
      <main class="content">
        <router-view />
      </main>
    </div>
    <BottomNav />
    <TaskDrawer />
  </div>
</template>

<script setup>
import TopBar from './components/TopBar.vue'
import Sidebar from './components/Sidebar.vue'
import BottomNav from './components/BottomNav.vue'
import TaskDrawer from './components/TaskDrawer.vue'
</script>
