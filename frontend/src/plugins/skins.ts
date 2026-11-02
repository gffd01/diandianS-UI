/**
 * 面板只保留两套浅色皮肤。背景与玻璃效果定义在 styles/skins.css，
 * 通过 <html data-skin="..."> 切换。
 */
export type SkinId = 'mesh' | 'mint'

export interface Skin {
  id: SkinId
  base: 'light' | 'dark'
  swatch: string // small gradient used as the menu preview chip
}

export const SKINS: Skin[] = [
  { id: 'mesh', base: 'light', swatch: 'linear-gradient(135deg,#c2e9fb,#fbc2eb)' },
  { id: 'mint', base: 'light', swatch: 'linear-gradient(135deg,#a7f3d0,#bae6fd)' },
]

export const DEFAULT_SKIN: SkinId = 'mint'

export function savedSkin(): Skin {
  const id = localStorage.getItem('skin') as SkinId | null
  const skin = SKINS.find((s) => s.id === id)
  if (skin) return skin

  // 升级前保存的极光、深海等旧值已不再可用，统一迁移为薄荷青。
  localStorage.setItem('skin', DEFAULT_SKIN)
  return SKINS.find((s) => s.id === DEFAULT_SKIN)!
}

/** Reflect the chosen skin onto <html> so styles/skins.css picks its gradient. */
export function applySkinAttr(id: SkinId): void {
  document.documentElement.setAttribute('data-skin', id)
}
