import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '../store/auth'
import Files from '../pages/Files.vue'
import Upload from '../pages/Upload.vue'
import FileDetail from '../pages/FileDetail.vue'
import Login from '../pages/Login.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'files', component: Files },
    { path: '/upload', name: 'upload', component: Upload },
    { path: '/file/:id', name: 'detail', component: FileDetail },
    { path: '/login', name: 'login', component: Login },
  ],
})

router.beforeEach((to) => {
  if (to.path === '/login') return true
  const token = getAccessToken()
  if (!token) return { path: '/login', query: { redirect: to.fullPath } }
  return true
})

export default router
