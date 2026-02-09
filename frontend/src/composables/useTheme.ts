import { ref } from 'vue'

const isDark = ref(false)

export function useTheme() {
  function initTheme() {
    const saved = localStorage.getItem('vmcat-theme')
    if (saved === 'dark' || (!saved && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      isDark.value = true
      document.documentElement.classList.add('dark')
    }
  }

  function toggle() {
    isDark.value = !isDark.value
    if (isDark.value) {
      document.documentElement.classList.add('dark')
      localStorage.setItem('vmcat-theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      localStorage.setItem('vmcat-theme', 'light')
    }
  }

  return { isDark, initTheme, toggle }
}
