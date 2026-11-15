<template>
  <AppLayout full-bleed>
    <div class="image-playground-page" :aria-busy="status === 'loading'">
      <div ref="workspace" class="h-full min-w-0" v-show="status === 'ready'"></div>
      <div v-if="status !== 'ready'" class="flex h-full flex-col items-center justify-center gap-3 px-4 text-center text-sm">
        <p v-if="status === 'loading'" role="status">{{ t('common.loading') }}</p>
        <template v-else>
          <p role="alert">{{ t('imagePlayground.loadError') }}</p>
          <button type="button" class="btn btn-secondary" @click="loadPlayground">{{ t('imagePlayground.retry') }}</button>
        </template>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AUTH_TOKENS_REFRESHED_EVENT } from '@/api/tokenRefresh'
import { useAdminSettingsStore, useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import { findImagePlaygroundMenuItem, IMAGE_PLAYGROUND_PATH } from '@/utils/imagePlayground'

interface PlaygroundModule {
  mountImagePlayground: (container: HTMLElement, options: {
    getToken: () => string
    subscribeToToken: (listener: () => void) => () => void
    refreshToken: (rejectedToken: string) => Promise<void>
    navigate: (path: string) => void
  }) => () => void
}

// The release build stages the native bundle before Vite assigns content hashes.
const loadModule = import.meta.glob<PlaygroundModule>('/src/generated/image-playground/image-playground.js')
const loadStyle = import.meta.glob<string>('/src/generated/image-playground/image-playground.css', {
  import: 'default',
  query: '?url',
})
const modulePath = '/src/generated/image-playground/image-playground.js'
const stylePath = '/src/generated/image-playground/image-playground.css'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const adminSettingsStore = useAdminSettingsStore()
const workspace = ref<HTMLElement | null>(null)
const status = ref<'loading' | 'ready' | 'error'>('loading')
let unmountPlayground: (() => void) | undefined
let loadController: AbortController | undefined

function loadStylesheet(href: string, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = href
    link.addEventListener('load', () => resolve(), { once: true, signal })
    link.addEventListener('error', () => reject(new Error('Playground stylesheet failed to load')), { once: true, signal })
    signal.addEventListener('abort', () => {
      link.remove()
      reject(signal.reason)
    }, { once: true })
    document.head.append(link)
  })
}

async function loadPlayground() {
  stopPlayground()
  if (!workspace.value || !authStore.isAuthenticated) return

  status.value = 'loading'
  const controller = new AbortController()
  loadController = controller
  try {
    if (!loadModule[modulePath] || !loadStyle[stylePath]) throw new Error('Playground assets are missing')

    const [{ mountImagePlayground }, stylesheetUrl] = await Promise.all([
      loadModule[modulePath](),
      loadStyle[stylePath](),
    ])
    if (controller.signal.aborted) return
    await loadStylesheet(stylesheetUrl, controller.signal)
    if (controller.signal.aborted || !workspace.value || !authStore.isAuthenticated) return

    unmountPlayground = mountImagePlayground(workspace.value, {
      getToken: authStore.getCurrentSessionToken,
      subscribeToToken: (listener) => watch([() => authStore.token, () => authStore.user?.id], listener, { flush: 'sync' }),
      refreshToken: authStore.refreshRejectedAccessToken,
      navigate: (path) => { void router.push(path) },
    })
    status.value = 'ready'
  } catch {
    if (controller.signal.aborted) return
    controller.abort()
    status.value = 'error'
  }
}

function stopPlayground() {
  unmountPlayground?.()
  unmountPlayground = undefined
  loadController?.abort()
  loadController = undefined
}

watch(() => authStore.isAuthenticated, (isAuthenticated) => {
  if (!isAuthenticated) {
    stopPlayground()
    void router.replace({ path: '/login', query: { redirect: IMAGE_PLAYGROUND_PATH } })
  }
})

watch(() => {
  const userItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  if (appStore.backendModeEnabled && !authStore.isAdmin) return false
  return !!findImagePlaygroundMenuItem(userItems, 'user') ||
    (authStore.isAdmin && !!findImagePlaygroundMenuItem(adminSettingsStore.customMenuItems, 'admin'))
}, (enabled) => {
  if (!enabled && authStore.isAuthenticated) {
    stopPlayground()
    void router.replace(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
  }
})

function handleStorage(event: StorageEvent) {
  if (event.storageArea === localStorage) authStore.syncPersistedSession()
}

onMounted(() => {
  window.addEventListener('storage', handleStorage)
  window.addEventListener(AUTH_TOKENS_REFRESHED_EVENT, authStore.syncPersistedSession)
  authStore.syncPersistedSession()
  void loadPlayground()
})
onBeforeUnmount(() => {
  window.removeEventListener('storage', handleStorage)
  window.removeEventListener(AUTH_TOKENS_REFRESHED_EVENT, authStore.syncPersistedSession)
  stopPlayground()
})
</script>

<style scoped>
.image-playground-page {
  height: var(--llmr-content-height, calc(100vh - 4rem));
  min-width: 0;
}

@container llmr-image-playground (max-width: 700px) {
  .image-playground-page :deep(.control-section:not(.prompt-section) .disclosure-inner) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .image-playground-page :deep(.control-section:not(.prompt-section) .field) {
    min-width: 0;
  }
}
</style>
