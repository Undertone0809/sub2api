import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CustomMenuItem } from '@/types'

const { authStore, appStore, adminSettingsStore } = vi.hoisted(() => ({
  authStore: { checkAuth: vi.fn(), isAuthenticated: false, isAdmin: false, isSimpleMode: false },
  appStore: {
    siteName: 'LLM Route',
    backendModeEnabled: false,
    publicSettingsLoaded: true,
    cachedPublicSettings: { custom_menu_items: [] as CustomMenuItem[] },
    fetchPublicSettings: vi.fn(),
  },
  adminSettingsStore: { customMenuItems: [] as CustomMenuItem[], fetch: vi.fn() },
}))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => adminSettingsStore }))
vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({ startNavigation: vi.fn(), endNavigation: vi.fn() }),
}))
vi.mock('@/composables/useRoutePrefetch', () => ({ useRoutePrefetch: () => ({ triggerPrefetch: vi.fn() }) }))
vi.mock('@/views/auth/LoginView.vue', () => ({ default: {} }))
vi.mock('@/views/user/DashboardView.vue', () => ({ default: {} }))
vi.mock('@/views/user/ImagePlaygroundView.vue', () => ({ default: {} }))

const publicPlayground: CustomMenuItem = {
  id: 'image-playground',
  label: 'AI 生图',
  icon_svg: '',
  url: 'https://llmroute.example/image-playground/',
  sort_order: 5,
  visibility: 'user',
}

describe('native AI image route', () => {
  beforeEach(async () => {
    vi.resetModules()
    const { initI18n } = await import('@/i18n')
    await initI18n()
    vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings.custom_menu_items = []
    appStore.fetchPublicSettings.mockReset()
    adminSettingsStore.customMenuItems = []
    adminSettingsStore.fetch.mockReset()
  })

  it.each(['/custom/image-playground', '/custom/%69mage-playground', '/ai-image'])(
    'strips legacy credentials from %s before returning to login', async (path) => {
      const { default: router } = await import('@/router')
      await router.push(`${path}?token=old-token&user_id=42#previous`)
      expect(router.currentRoute.value.fullPath).toBe('/login?redirect=/ai-image')
      expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
      expect(adminSettingsStore.fetch).not.toHaveBeenCalled()
    },
  )

  it('loads a configured user entry and uses its menu label for the title', async () => {
    authStore.isAuthenticated = true
    appStore.publicSettingsLoaded = false
    appStore.fetchPublicSettings.mockImplementationOnce(async () => {
      appStore.cachedPublicSettings.custom_menu_items = [publicPlayground]
      appStore.publicSettingsLoaded = true
    })
    const { default: router } = await import('@/router')
    await router.push('/custom/image-playground')
    expect(router.currentRoute.value.name).toBe('ImagePlayground')
    expect(document.title).toBe('AI 生图 - LLM Route')
    expect(adminSettingsStore.fetch).not.toHaveBeenCalled()
  })

  it('denies direct navigation when the item is absent from public settings', async () => {
    authStore.isAuthenticated = true
    adminSettingsStore.customMenuItems = [{ ...publicPlayground, visibility: 'admin' }]
    const { default: router } = await import('@/router')
    await router.push('/ai-image')
    expect(router.currentRoute.value.path).toBe('/dashboard')
    expect(adminSettingsStore.fetch).not.toHaveBeenCalled()
  })

  it('honors an admin-only item after loading admin settings', async () => {
    authStore.isAuthenticated = true
    authStore.isAdmin = true
    adminSettingsStore.fetch.mockImplementationOnce(async () => {
      adminSettingsStore.customMenuItems = [{ ...publicPlayground, visibility: 'admin' }]
    })
    const { default: router } = await import('@/router')
    await router.push('/ai-image')
    expect(router.currentRoute.value.name).toBe('ImagePlayground')
    expect(adminSettingsStore.fetch).toHaveBeenCalledOnce()
  })
})
