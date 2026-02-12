import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './assets/index.css'
import App from './App.vue'
import router from './router'
import { useSettingsStore } from '@/stores/settings.ts'

const REQUIRED_ENV_VARS = ['VITE_API_URL', 'VITE_BOT_USERNAME']

const missingVars = REQUIRED_ENV_VARS.filter((key) => !import.meta.env[key])

if (missingVars.length > 0) {
    const errorMsg = `Missing environment variables: ${missingVars.join(', ')}`
    console.error(errorMsg)
    document.body.innerHTML = `
    <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100vh; background: #020817; color: #f8fafc; font-family: sans-serif; text-align: center; padding: 2rem;">
      <h1 style="color: #ef4444; margin-bottom: 1rem;">Configuration Error</h1>
      <p style="margin-bottom: 2rem;">${errorMsg}</p>
      <code style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; text-align: left;">
        Please check your environment variables.
      </code>
    </div>
  `
} else {
    const app = createApp(App)
    const pinia = createPinia()

    app.use(pinia)
    app.use(router)

    const settingsStore = useSettingsStore()
    settingsStore.fetchSettings()

    app.mount('#app')
}
