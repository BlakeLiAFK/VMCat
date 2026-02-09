import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/Dashboard.vue'),
    },
    {
      path: '/host/:id',
      name: 'host-detail',
      component: () => import('@/views/HostDetail.vue'),
    },
    {
      path: '/host/:id/vm/:name',
      name: 'vm-detail',
      component: () => import('@/views/VMDetail.vue'),
    },
    {
      path: '/host/:id/terminal',
      name: 'terminal',
      component: () => import('@/views/SSHTerminal.vue'),
    },
    {
      path: '/host/:id/vm/:name/vnc',
      name: 'vnc',
      component: () => import('@/views/VNCViewer.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/Settings.vue'),
    },
  ],
})

export default router
