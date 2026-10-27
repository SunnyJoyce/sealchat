import type { UIRuntimeSurface, UISurface } from './surfaces'

export interface UICustomizationContext {
  surface: UIRuntimeSurface
  worldId: string
  channelId: string
  slug: string
  observer: boolean
  publicSurface: boolean
}

export type UICustomizationScopeType = 'platform' | 'world'

export interface UICustomizationPackage {
  id: string
  scopeType: UICustomizationScopeType
  scopeId: string
  name: string
  description: string
  enabled: boolean
  sortOrder: number
  surfaces: UISurface[]
  css: string
  js: string
  jsHash: string
  createdBy: string
  updatedBy: string
  createdAt: number
  updatedAt: number
}

export interface UICustomizationPolicy {
  allowWorldCss: boolean
  allowWorldJs: boolean
  allowPublicWorldCss: boolean
}

export type UICustomizationPackageInput = Partial<Pick<UICustomizationPackage, 'name' | 'description' | 'enabled' | 'surfaces' | 'css' | 'js'>>

// js is empty whenever the server decided the script may not run here.
// requiresTrust marks world scripts that also need per-browser trust.
export type RuntimePackage = Pick<UICustomizationPackage, 'id' | 'scopeType' | 'name' | 'sortOrder' | 'createdAt' | 'css'>
  & { js?: string; jsHash?: string; requiresTrust?: boolean }
export interface UICustomizationRuntimeResult {
  surface: UIRuntimeSurface
  worldId: string
  channelId: string
  packages: RuntimePackage[]
}
