<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  activeView: {
    type: String,
    default: 'dashboard',
  },
  username: {
    type: String,
    default: '',
  },
  avatarUrl: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['navigate', 'logout', 'change-password', 'about'])

const avatarLetter = computed(() => props.username?.trim().charAt(0).toUpperCase() || '?')

const ICON_EMOJI = {
  dashboard: '📊',
  purchases: '🛍️',
  incomes: '💰',
  budgets: '🎯',
  accounts: '🏦',
  categories: '🏷️',
  import: '📥',
  'import-transactions': '🧾',
  reports: '📈',
  users: '👥',
  preferences: '⚙️',
  reconciliation: '✅',
  about: 'ℹ️',
}

const ICON_COLORS = {
  dashboard: '#7aa2f7',
  purchases: '#f0a95e',
  incomes: '#5ecb8f',
  budgets: '#b18cf0',
  accounts: '#5cc8a0',
  categories: '#f07aa5',
  import: '#5cc8d1',
  'import-transactions': '#5cc8d1',
  reports: '#e0c15c',
  users: '#f0796a',
  preferences: '#9aa5b1',
  reconciliation: '#4fd1c5',
  about: '#8fa8a0',
}

const navItems = [
  { key: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
  { key: 'purchases', label: 'Achats', icon: 'purchases' },
  { key: 'incomes', label: 'Revenus', icon: 'incomes' },
  { key: 'budgets', label: 'Budgets', icon: 'budgets' },
  { key: 'reconciliation', label: 'Pointage', icon: 'reconciliation' },
  { key: 'accounts', label: 'Comptes', icon: 'accounts' },
  { key: 'categories', label: 'Catégories', icon: 'categories' },
  {
    key: 'import-group',
    label: 'Import',
    icon: 'import',
    isGroup: true,
    children: [
      { key: 'import', label: 'Achats / Revenus', icon: 'import-transactions' },
      { key: 'import-categories', label: 'Catégories', icon: 'categories' },
    ],
  },
  { key: 'reports', label: 'Éditions', icon: 'reports' },
  { key: 'users', label: 'Utilisateurs', icon: 'users' },
  { key: 'preferences', label: 'Préférences', icon: 'preferences' },
  { key: 'about', label: 'À propos', icon: 'about', action: 'about' },
]

function handleNavClick(item) {
  if (item.action === 'about') {
    emit('about')
  } else {
    emit('navigate', item.key)
  }

  closeMobileMenu()
}

function handleChildNavClick(childKey) {
  emit('navigate', childKey)
  closeMobileMenu()
}

function iconEmoji(name) {
  return ICON_EMOJI[name] || '•'
}

function iconColor(name) {
  return ICON_COLORS[name] || '#8fa8a0'
}

function shade(hex, percent) {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const target = percent < 0 ? 0 : 255
  const p = Math.abs(percent)

  const mix = (channel) => Math.round((target - channel) * p) + channel

  return `rgb(${mix(r)}, ${mix(g)}, ${mix(b)})`
}

function iconGradient(name) {
  const base = iconColor(name)
  const light = shade(base, 0.4)
  const dark = shade(base, -0.3)
  return `linear-gradient(150deg, ${light} 0%, ${base} 55%, ${dark} 100%)`
}

const SUBMENU_STORAGE_KEY = 'sidebarImportSubmenuOpen'
const isImportSubmenuOpen = ref(localStorage.getItem(SUBMENU_STORAGE_KEY) !== '0')

function toggleImportSubmenu() {
  isImportSubmenuOpen.value = !isImportSubmenuOpen.value
  localStorage.setItem(SUBMENU_STORAGE_KEY, isImportSubmenuOpen.value ? '1' : '0')
}

const WIDTH_STORAGE_KEY = 'sidebarWidth'
const COLLAPSED_STORAGE_KEY = 'sidebarCollapsed'
const MIN_WIDTH = 180
const MAX_WIDTH = 340
const DEFAULT_WIDTH = 208
const COLLAPSED_WIDTH = 60

function clampWidth(width) {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, width))
}

const storedWidth = Number(localStorage.getItem(WIDTH_STORAGE_KEY))
const sidebarWidth = ref(clampWidth(storedWidth > 0 ? storedWidth : DEFAULT_WIDTH))
const isCollapsed = ref(localStorage.getItem(COLLAPSED_STORAGE_KEY) === '1')
const isResizing = ref(false)

// En dessous de 920px, la barre latérale n'est plus affichée en permanence à
// côté du contenu (elle ne laisserait presque plus de place à celui-ci) —
// elle devient un tiroir en recouvrement, ouvert/fermé via le bouton
// hamburger, indépendant du réglage "réduite/redimensionnée" du mode bureau.
const MOBILE_BREAKPOINT = '(max-width: 920px)'
const mobileQuery = window.matchMedia(MOBILE_BREAKPOINT)
const isMobile = ref(mobileQuery.matches)
const isMobileMenuOpen = ref(false)

function handleMobileQueryChange(event) {
  isMobile.value = event.matches
  if (!isMobile.value) {
    isMobileMenuOpen.value = false
  }
}

onMounted(() => {
  mobileQuery.addEventListener('change', handleMobileQueryChange)
})

onBeforeUnmount(() => {
  mobileQuery.removeEventListener('change', handleMobileQueryChange)
})

function openMobileMenu() {
  isMobileMenuOpen.value = true
}

function closeMobileMenu() {
  if (isMobile.value) {
    isMobileMenuOpen.value = false
  }
}

// Empêche le contenu derrière le tiroir de scroller pendant qu'il est ouvert.
watch(isMobileMenuOpen, (isOpen) => {
  document.body.style.overflow = isOpen ? 'hidden' : ''
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

// Le mode "réduit" (icônes seules) est une commodité de bureau pour gagner
// de la place à côté du contenu — sur mobile, le tiroir est soit fermé
// (hors écran), soit ouvert en plein avec les libellés : jamais réduit.
const effectiveCollapsed = computed(() => isCollapsed.value && !isMobile.value)

const currentWidth = computed(() => {
  if (isMobile.value) return sidebarWidth.value
  return isCollapsed.value ? COLLAPSED_WIDTH : sidebarWidth.value
})

function toggleCollapsed() {
  isCollapsed.value = !isCollapsed.value
  localStorage.setItem(COLLAPSED_STORAGE_KEY, isCollapsed.value ? '1' : '0')
}

function handleResizeMove(event) {
  sidebarWidth.value = clampWidth(event.clientX)
}

function stopResize() {
  if (!isResizing.value) return

  isResizing.value = false
  document.body.style.removeProperty('cursor')
  document.body.style.removeProperty('user-select')
  window.removeEventListener('mousemove', handleResizeMove)
  window.removeEventListener('mouseup', stopResize)
  localStorage.setItem(WIDTH_STORAGE_KEY, String(sidebarWidth.value))
}

function resetWidth() {
  sidebarWidth.value = DEFAULT_WIDTH
  localStorage.setItem(WIDTH_STORAGE_KEY, String(DEFAULT_WIDTH))
}

function startResize(event) {
  if (isCollapsed.value) return

  isResizing.value = true
  event.preventDefault()
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', handleResizeMove)
  window.addEventListener('mouseup', stopResize)
}

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleResizeMove)
  window.removeEventListener('mouseup', stopResize)
})
</script>

<template>
  <button
    v-if="isMobile"
    class="mobile-menu-toggle"
    type="button"
    aria-label="Ouvrir le menu"
    @click="openMobileMenu"
  >
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M4 6h16M4 12h16M4 18h16"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
      />
    </svg>
  </button>

  <div
    v-if="isMobile && isMobileMenuOpen"
    class="mobile-backdrop"
    @click="closeMobileMenu"
  ></div>

  <aside
    class="sidebar"
    :class="{
      'is-collapsed': effectiveCollapsed,
      'is-resizing': isResizing,
      'is-mobile': isMobile,
      'is-mobile-open': isMobile && isMobileMenuOpen,
    }"
    :style="{ width: currentWidth + 'px', minWidth: currentWidth + 'px' }"
  >
    <div class="sidebar-scroll">
      <div class="brand">
        <div class="brand-mark">F</div>
        <div v-if="!effectiveCollapsed">
          <strong>Finance UI</strong>
          <p>Personal purchases</p>
        </div>

        <button
          v-if="isMobile"
          class="mobile-close-btn"
          type="button"
          aria-label="Fermer le menu"
          @click="closeMobileMenu"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M6 6l12 12M18 6L6 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
          </svg>
        </button>
      </div>

      <nav class="sidebar-nav" aria-label="Navigation principale">
        <template v-for="item in navItems" :key="item.key">
          <template v-if="item.isGroup">
            <div class="nav-group" :class="{ 'nav-group-collapsed': effectiveCollapsed }">
              <button
                class="nav-item nav-group-toggle"
                :class="{ 'nav-item-collapsed': effectiveCollapsed }"
                type="button"
                :aria-label="item.label"
                @click="toggleImportSubmenu"
              >
                <span class="nav-icon-chip" :style="{ background: iconGradient(item.icon) }">
                  <span class="nav-emoji">{{ iconEmoji(item.icon) }}</span>
                </span>
                <span v-if="!effectiveCollapsed" class="nav-group-toggle-label">{{ item.label }}</span>
                <span v-else class="nav-tooltip">{{ item.label }}</span>
                <svg
                  v-if="!effectiveCollapsed"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                  class="nav-group-chevron"
                  :class="{ 'is-open': isImportSubmenuOpen }"
                >
                  <path
                    d="M9 6l6 6-6 6"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>

              <div
                v-if="isImportSubmenuOpen"
                class="nav-children-rail"
                :class="{ 'has-rail': !effectiveCollapsed }"
              >
                <button
                  v-for="child in item.children"
                  :key="child.key"
                  class="nav-item nav-item-child"
                  :class="{ active: activeView === child.key, 'nav-item-collapsed': effectiveCollapsed }"
                  type="button"
                  :aria-label="`${item.label} — ${child.label}`"
                  @click="handleChildNavClick(child.key)"
                >
                  <span class="nav-icon-chip" :style="{ background: iconGradient(child.icon) }">
                    <span class="nav-emoji">{{ iconEmoji(child.icon) }}</span>
                  </span>
                  <span v-if="!effectiveCollapsed">{{ child.label }}</span>
                  <span v-else class="nav-tooltip">{{ item.label }} — {{ child.label }}</span>
                </button>
              </div>
            </div>
          </template>

          <button
            v-else
            class="nav-item"
            :class="{ active: activeView === item.key, 'nav-item-collapsed': effectiveCollapsed }"
            type="button"
            :aria-label="item.label"
            @click="handleNavClick(item)"
          >
            <span class="nav-icon-chip" :style="{ background: iconGradient(item.icon) }">
              <span class="nav-emoji">{{ iconEmoji(item.icon) }}</span>
            </span>
            <span v-if="!effectiveCollapsed">{{ item.label }}</span>
            <span v-else class="nav-tooltip">{{ item.label }}</span>
          </button>
        </template>
      </nav>
    </div>

    <div class="sidebar-footer">
      <div class="profile-card" :class="{ 'profile-card-collapsed': effectiveCollapsed }">
        <div class="avatar" :aria-label="username">
          <span v-if="avatarUrl" class="avatar-photo"><img :src="avatarUrl" alt="" /></span>
          <template v-else>{{ avatarLetter }}</template>
          <span v-if="effectiveCollapsed" class="nav-tooltip">{{ username || 'Compte' }}</span>
        </div>

        <template v-if="!effectiveCollapsed">
          <div class="profile-info">
            <strong>{{ username || 'Compte' }}</strong>
            <p>Compte principal</p>
          </div>
          <div class="profile-actions">
            <button
              class="icon-action-btn"
              type="button"
              title="Modifier le mot de passe"
              aria-label="Modifier le mot de passe"
              @click="$emit('change-password')"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M12 17a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM12 13v-2M7 9V7a5 5 0 0 1 10 0v2"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <rect
                  x="5"
                  y="9"
                  width="14"
                  height="11"
                  rx="2.5"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                />
              </svg>
            </button>

            <button
              class="icon-action-btn icon-action-btn-danger"
              type="button"
              title="Se déconnecter"
              aria-label="Se déconnecter"
              @click="$emit('logout')"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M15 17l5-5-5-5M20 12H9M12 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h6"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </div>
        </template>
      </div>

      <button
        v-if="!isMobile"
        class="collapse-toggle"
        type="button"
        :title="isCollapsed ? 'Afficher la barre latérale' : 'Masquer la barre latérale'"
        :aria-label="isCollapsed ? 'Afficher la barre latérale' : 'Masquer la barre latérale'"
        @click="toggleCollapsed"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" :class="{ 'is-flipped': isCollapsed }">
          <path
            d="M15 5l-7 7 7 7"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <span v-if="isCollapsed">Afficher</span>
        <span v-else>Masquer</span>
      </button>
    </div>

    <div
      v-if="!effectiveCollapsed && !isMobile"
      class="resize-handle"
      role="separator"
      aria-orientation="vertical"
      aria-label="Redimensionner la barre latérale"
      @mousedown="startResize"
      @dblclick="resetWidth"
    ></div>
  </aside>
</template>

<style scoped>
.sidebar {
  position: relative;
  min-height: 100vh;
  padding: 1.2rem 1rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background: var(--sidebar-bg);
  border-right: 1px solid rgba(var(--tint-rgb), 0.05);
}

.sidebar:not(.is-resizing) {
  transition: width 160ms ease, min-width 160ms ease;
}

.sidebar.is-collapsed {
  padding-inline: 0.6rem;
}

.sidebar-scroll {
  overflow-x: hidden;
}

.sidebar.is-collapsed .sidebar-scroll {
  overflow: visible;
}

.mobile-menu-toggle {
  display: none;
}

.mobile-close-btn {
  display: none;
}

.mobile-backdrop {
  display: none;
}

/* En dessous de 920px, la barre latérale quitte le flux normal (elle
   prendrait presque toute la largeur de l'écran) et devient un tiroir en
   recouvrement : caché hors écran par défaut, glissé au premier plan à
   l'ouverture. Le bouton hamburger qui l'ouvre reste toujours visible,
   lui, puisqu'il vit en dehors de la barre (donc jamais caché hors écran
   avec elle). */
@media (max-width: 920px) {
  .mobile-menu-toggle {
    display: grid;
    place-items: center;
    position: fixed;
    top: 14px;
    left: 14px;
    z-index: 110;
    width: 42px;
    height: 42px;
    border: 1px solid rgba(var(--tint-rgb), 0.1);
    border-radius: 12px;
    background: var(--sidebar-bg);
    color: var(--text, #eef1f3);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.28);
    cursor: pointer;
  }

  .mobile-menu-toggle svg {
    width: 20px;
    height: 20px;
  }

  .mobile-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.5);
    animation: mobile-backdrop-fade-in 160ms ease;
  }

  .mobile-close-btn {
    display: grid;
    place-items: center;
    margin-left: auto;
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    border: none;
    border-radius: 10px;
    background: rgba(var(--tint-rgb), 0.06);
    color: var(--text, #eef1f3);
    cursor: pointer;
  }

  .mobile-close-btn svg {
    width: 16px;
    height: 16px;
  }

  .sidebar.is-mobile {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 105;
    min-height: 100dvh;
    height: 100dvh;
    /* Filet de sécurité sur un très petit téléphone : la largeur du tiroir
       vient du réglage bureau (jusqu'à 340px), qui peut dépasser un écran
       étroit — jamais plus de 85% de la largeur visible. */
    max-width: 85vw;
    transform: translateX(-100%);
    box-shadow: 12px 0 40px rgba(0, 0, 0, 0.35);
    transition: transform 220ms ease;
  }

  .sidebar.is-mobile.is-mobile-open {
    transform: translateX(0);
  }
}

@keyframes mobile-backdrop-fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  margin-bottom: 2rem;
}

.is-collapsed .brand {
  justify-content: center;
}

.brand-mark {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm, 12px);
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent, #3b82f6), var(--accent-blue, #2f6fe0));
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.35);
  color: #ffffff;
  font-weight: 700;
  flex-shrink: 0;
}

.brand strong {
  display: block;
  color: var(--text);
  font-size: 0.98rem;
}

.brand p {
  margin-top: 0.2rem;
  color: rgba(var(--tint-rgb), 0.56);
  font-size: 0.84rem;
}

.sidebar-nav {
  display: grid;
  gap: 0.45rem;
}

.nav-group {
  display: contents;
}

.nav-group.nav-group-collapsed {
  display: grid;
  gap: 0.35rem;
  padding: 0.35rem 0.2rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.05);
  border: 1px solid rgba(var(--tint-rgb), 0.08);
}

.nav-group-toggle-label {
  flex: 1;
}

.nav-group-chevron {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  color: rgba(var(--tint-rgb), 0.4);
  transition: transform 160ms ease;
}

.nav-group-chevron.is-open {
  transform: rotate(90deg);
}

.nav-children-rail {
  position: relative;
  display: grid;
  gap: 0.45rem;
}

.nav-children-rail.has-rail {
  padding-left: 1.55rem;
  margin-left: 0.35rem;
}

.nav-children-rail.has-rail::before {
  content: '';
  position: absolute;
  top: 0.2rem;
  bottom: 0.2rem;
  left: 0;
  width: 1px;
  background: rgba(var(--tint-rgb), 0.1);
}

.nav-item-child {
  gap: 0.55rem;
  color: rgba(var(--tint-rgb), 0.5);
  font-size: 0.74rem;
  font-weight: 500;
}

.nav-item-child .nav-icon-chip {
  width: 26px;
  height: 26px;
}

.nav-item-child .nav-emoji {
  font-size: 13px;
}

.nav-item {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.7rem;
  border: none;
  border-radius: 14px;
  padding: 0.95rem 1rem;
  background: transparent;
  color: rgba(var(--tint-rgb), 0.68);
  text-align: left;
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    background 160ms ease,
    color 160ms ease,
    transform 160ms ease;
}

.nav-tooltip {
  position: absolute;
  left: calc(100% + 10px);
  top: 50%;
  transform: translateY(-50%) translateX(-4px);
  z-index: 30;
  padding: 0.4rem 0.7rem;
  border-radius: 8px;
  background: var(--modal-bg, #1c2024);
  color: var(--text, #eef1f3);
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.28);
  opacity: 0;
  pointer-events: none;
  transition: opacity 140ms ease, transform 140ms ease;
}

.nav-item:hover .nav-tooltip,
.nav-item:focus-visible .nav-tooltip {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
}

.nav-icon-chip {
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border-radius: 9px;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.3),
    inset 0 -6px 10px rgba(0, 0, 0, 0.18),
    0 2px 5px rgba(0, 0, 0, 0.28);
  transition: transform 160ms ease;
}

.nav-item:hover .nav-icon-chip {
  transform: scale(1.06) translateY(-1px);
}

.nav-emoji {
  font-size: 15px;
  line-height: 1;
  filter: drop-shadow(0 1px 1px rgba(0, 0, 0, 0.25));
}

.nav-item:hover {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
}

.nav-item:focus-visible {
  outline: 2px solid rgba(199, 217, 209, 0.9);
  outline-offset: 2px;
}

.nav-item.active {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.18), rgba(59, 130, 246, 0.06));
  color: var(--text);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.3);
}

.nav-item-collapsed {
  display: grid;
  place-items: center;
  padding: 0.7rem 0;
}

.sidebar-footer {
  padding-top: 1rem;
  display: grid;
  gap: 0.6rem;
}

.collapse-toggle {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  width: 100%;
  height: 34px;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 10px;
  background: rgba(var(--tint-rgb), 0.03);
  color: rgba(var(--tint-rgb), 0.6);
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}

.collapse-toggle:hover {
  background: rgba(var(--tint-rgb), 0.08);
  color: var(--text);
}

.collapse-toggle svg {
  width: 14px;
  height: 14px;
  transition: transform 160ms ease;
  flex-shrink: 0;
}

.collapse-toggle svg.is-flipped {
  transform: rotate(180deg);
}

.resize-handle {
  position: absolute;
  top: 0;
  right: -3px;
  width: 6px;
  height: 100%;
  cursor: col-resize;
  z-index: 5;
}

.resize-handle:hover,
.is-resizing .resize-handle {
  background: rgba(114, 137, 152, 0.35);
}

.profile-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.85rem 0.9rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.profile-card-collapsed {
  justify-content: center;
  padding: 0.6rem;
}

.avatar {
  position: relative;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  overflow: visible;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.1);
  color: var(--text);
  font-weight: 700;
  font-size: 0.88rem;
  flex-shrink: 0;
}

.avatar-photo {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  overflow: hidden;
}

.avatar-photo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar:hover .nav-tooltip,
.avatar:focus-visible .nav-tooltip {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
}

.profile-info {
  min-width: 0;
  flex: 1;
}

.profile-card strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text);
  font-size: 0.92rem;
}

.profile-card p {
  margin-top: 0.15rem;
  color: rgba(var(--tint-rgb), 0.54);
  font-size: 0.78rem;
}

.profile-actions {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  flex-shrink: 0;
}

.icon-action-btn {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 9px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.03);
  color: rgba(var(--tint-rgb), 0.6);
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease, transform 140ms ease;
}

.icon-action-btn:hover {
  background: rgba(var(--tint-rgb), 0.09);
  color: var(--text);
  transform: translateY(-1px);
}

.icon-action-btn-danger:hover {
  background: rgba(220, 38, 38, 0.14);
  color: var(--negative-text);
}

.icon-action-btn svg {
  width: 14px;
  height: 14px;
}
</style>
