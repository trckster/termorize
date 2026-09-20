import { ref } from 'vue'

// Palette + light/dark theming, persisted in localStorage. Frontend-only — the
// choice never touches the backend. The CSS for each palette lives in
// assets/index.css (scoped by `data-theme` on <html>); dark mode is the `.dark`
// class. The inline script in index.html applies both pre-paint to avoid a flash.

export type PaletteId =
    | 'evergreen'
    | 'sage'
    | 'emerald'
    | 'citrine'
    | 'tangerine'
    | 'terracotta'
    | 'iris'
    | 'amethyst'
    | 'plum'

export interface ThemeOption {
    id: PaletteId
    /** Display name — a proper noun, intentionally not translated. */
    name: string
}

export const THEME_OPTIONS: ThemeOption[] = [
    { id: 'evergreen', name: 'Evergreen' },
    { id: 'sage', name: 'Sage' },
    { id: 'emerald', name: 'Emerald' },
    { id: 'citrine', name: 'Citrine' },
    { id: 'tangerine', name: 'Tangerine' },
    { id: 'terracotta', name: 'Terracotta' },
    { id: 'iris', name: 'Iris' },
    { id: 'amethyst', name: 'Amethyst' },
    { id: 'plum', name: 'Plum' },
]

const PALETTE_IDS = THEME_OPTIONS.map((o) => o.id)
const DEFAULT_PALETTE: PaletteId = 'evergreen'
const PALETTE_KEY = 'palette'
const THEME_KEY = 'theme'

const palette = ref<PaletteId>(DEFAULT_PALETTE)
const isDark = ref(false)

const root = () => document.documentElement

const normalizePalette = (value: string | null): PaletteId =>
    PALETTE_IDS.includes(value as PaletteId) ? (value as PaletteId) : DEFAULT_PALETTE

const prefersDark = () => window.matchMedia('(prefers-color-scheme: dark)').matches

const applyPalette = () => root().setAttribute('data-theme', palette.value)

const applyDark = () => {
    root().classList.toggle('dark', isDark.value)
    root().style.colorScheme = isDark.value ? 'dark' : 'light'
}

/** True while the user has no explicit light/dark choice (i.e. follows the OS). */
export const followsSystemTheme = () => !localStorage.getItem(THEME_KEY)

/** Read persisted preferences and apply them. Safe to call more than once. */
export const initTheme = () => {
    palette.value = normalizePalette(localStorage.getItem(PALETTE_KEY))
    isDark.value = localStorage.getItem(THEME_KEY) === 'dark' || (followsSystemTheme() && prefersDark())
    applyPalette()
    applyDark()
}

export function useTheme() {
    const setPalette = (id: PaletteId) => {
        palette.value = id
        localStorage.setItem(PALETTE_KEY, id)
        applyPalette()
    }

    const setDark = (value: boolean) => {
        isDark.value = value
        localStorage.setItem(THEME_KEY, value ? 'dark' : 'light')
        applyDark()
    }

    const toggleDark = () => setDark(!isDark.value)

    /** Re-apply the OS preference, but only if the user hasn't chosen explicitly. */
    const syncSystemTheme = () => {
        if (!followsSystemTheme()) return
        isDark.value = prefersDark()
        applyDark()
    }

    return { palette, isDark, options: THEME_OPTIONS, setPalette, setDark, toggleDark, syncSystemTheme }
}
