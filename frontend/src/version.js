import { version } from '../package.json'

// Single source of truth for the app version shown in LoginView and the
// "À propos" panel — bump package.json's version to update both at once.
export const APP_VERSION = version

// __BUILD_TIME__ is injected by Vite (see vite.config.js `define`) as the
// ISO timestamp of when this bundle was built (or when the dev server started).
export const APP_BUILD_TIME = __BUILD_TIME__
