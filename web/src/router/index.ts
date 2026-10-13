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
    {
      path: '/profile',
      name: 'profile',
      redirect: (to) => ({ name: 'lobby', query: { panel: to.query.tab ? 'settings' : 'profile', section: to.query.tab } }),
    },
    {
      path: '/table/:id',
      name: 'table',
      component: () => import('../views/table/TableView.vue'),
      meta: { requiresAuth: true },
    },
    ...(import.meta.env.DEV
      ? [{ path: '/dev/sounds', name: 'dev-sounds', component: () => import('../views/dev/SoundsView.vue') }]
      : []),
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
