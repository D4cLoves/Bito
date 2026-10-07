import { createRouter, createWebHistory } from 'vue-router'
import LandingView from '../views/landing/LandingView.vue'

// Вход и регистрация живут в карточке на лендинге; старые адреса ведут туда.
const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'landing',
      component: LandingView,
    },
    {
      path: '/login',
      redirect: { name: 'landing', query: { auth: 'login' } },
    },
    {
      path: '/register',
      redirect: { name: 'landing' },
    },
    {
      path: '/lobby',
      name: 'lobby',
      component: () => import('../views/lobby/LobbyView.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    return { name: 'landing', query: { auth: 'login' } }
  }
  return true
})

export default router
