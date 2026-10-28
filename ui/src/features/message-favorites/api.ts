import { api } from '@/stores/_config'
import type { Message } from '@satorijs/protocol'

// 消息收藏 REST 契约的唯一归一化边界。字段语义见 doc/message-favorites-backend-design.md。

export type MessageFavoriteScope = 'personal' | 'world'
export type MessageFavoriteStatus = 'available' | 'deleted' | 'unavailable'
export type MessageFavoriteRejectReason = 'not_visible' | 'wrong_world' | 'whisper_not_allowed' | 'not_found' | string

export interface MessageFavoriteTag {
  id: string
  scope: MessageFavoriteScope
  worldId: string
  name: string
}

export interface MessageFavoriteItem {
  id: string
  scope: MessageFavoriteScope
  worldId: string
  channelId: string
  messageId: string
  sortOrder: number
  createdAt: string
  tags: MessageFavoriteTag[]
  status: MessageFavoriteStatus
  // 只有 available 时存在；deleted / unavailable 不携带旧正文。
  message: Message | null
}

export interface MessageFavoriteRef {
  id: string
  worldId: string
  messageId: string
}

export interface MessageFavoriteListQuery {
  scope: MessageFavoriteScope
  // world：收藏所属世界（必填）；personal：可选世界筛选。
  worldId: string
  q: string
  tagIds: string[]
}

export interface MessageFavoriteListResult {
  items: MessageFavoriteItem[]
  nextCursor: string
  canManageWorld: boolean
}

export interface MessageFavoriteBatchResult {
  created: MessageFavoriteRef[]
  existing: MessageFavoriteRef[]
  rejected: Array<{ messageId: string, reason: MessageFavoriteRejectReason }>
}

const normalizeScope = (raw: unknown): MessageFavoriteScope => (raw === 'world' ? 'world' : 'personal')

const normalizeTag = (raw: any): MessageFavoriteTag => ({
  id: String(raw?.id || ''),
  scope: normalizeScope(raw?.scope),
  worldId: String(raw?.worldId || ''),
  name: String(raw?.name || ''),
})

const normalizeStatus = (raw: unknown): MessageFavoriteStatus => (
  raw === 'available' || raw === 'deleted' ? raw : 'unavailable'
)

const normalizeItem = (raw: any): MessageFavoriteItem => {
  const status = normalizeStatus(raw?.status)
  return {
    id: String(raw?.id || ''),
    scope: normalizeScope(raw?.scope),
    worldId: String(raw?.worldId || ''),
    channelId: String(raw?.channelId || ''),
    messageId: String(raw?.messageId || ''),
    sortOrder: Number(raw?.sortOrder) || 0,
    createdAt: String(raw?.createdAt || ''),
    tags: Array.isArray(raw?.tags) ? raw.tags.map(normalizeTag) : [],
    status,
    message: status === 'available' && raw?.message && typeof raw.message === 'object' ? raw.message as Message : null,
  }
}

const normalizeRef = (raw: any): MessageFavoriteRef => ({
  id: String(raw?.id || ''),
  worldId: String(raw?.worldId || ''),
  messageId: String(raw?.messageId || ''),
})

const scopeParams = (scope: MessageFavoriteScope, worldId: string) => (
  scope === 'world' ? { scope, worldId } : { scope }
)

export const describeMessageFavoriteError = (error: unknown, fallback = '操作失败') => {
  const data = (error as any)?.response?.data
  const message = String(data?.message || data?.error || '').trim()
  return message || (error instanceof Error && error.message ? error.message : fallback)
}

export const fetchMessageFavorites = async (
  query: MessageFavoriteListQuery,
  options: { cursor?: string, limit?: number } = {},
): Promise<MessageFavoriteListResult> => {
  const params: Record<string, string | number> = { scope: query.scope }
  if (query.worldId) params.worldId = query.worldId
  if (query.q) params.q = query.q
  // 后端同时接受重复参数和逗号分隔；逗号形式避免 axios 数组序列化差异。
  if (query.tagIds.length) params.tagIds = query.tagIds.join(',')
  if (options.cursor) params.cursor = options.cursor
  if (options.limit) params.limit = options.limit
  const resp = await api.get('api/v1/message-favorites', { params })
  return {
    items: Array.isArray(resp.data?.items) ? resp.data.items.map(normalizeItem) : [],
    nextCursor: String(resp.data?.nextCursor || ''),
    canManageWorld: resp.data?.canManageWorld === true,
  }
}

export const batchCreateMessageFavorites = async (
  scope: MessageFavoriteScope,
  worldId: string,
  messageIds: string[],
): Promise<MessageFavoriteBatchResult> => {
  const resp = await api.post('api/v1/message-favorites/batch', { ...scopeParams(scope, worldId), messageIds })
  return {
    created: Array.isArray(resp.data?.created) ? resp.data.created.map(normalizeRef) : [],
    existing: Array.isArray(resp.data?.existing) ? resp.data.existing.map(normalizeRef) : [],
    rejected: Array.isArray(resp.data?.rejected)
      ? resp.data.rejected.map((item: any) => ({ messageId: String(item?.messageId || ''), reason: String(item?.reason || '') }))
      : [],
  }
}

export const deleteMessageFavorite = async (id: string) => {
  await api.delete(`api/v1/message-favorites/${encodeURIComponent(id)}`)
}

// afterId 是目标前驱，beforeId 是目标后继；至少提供一个。
export const moveMessageFavorite = async (id: string, neighbor: { afterId?: string, beforeId?: string }) => {
  await api.post(`api/v1/message-favorites/${encodeURIComponent(id)}/move`, neighbor)
}

export const replaceMessageFavoriteTags = async (id: string, tagIds: string[]): Promise<MessageFavoriteTag[]> => {
  const resp = await api.put(`api/v1/message-favorites/${encodeURIComponent(id)}/tags`, { tagIds })
  return Array.isArray(resp.data?.tags) ? resp.data.tags.map(normalizeTag) : []
}

export const fetchMessageFavoriteTags = async (scope: MessageFavoriteScope, worldId: string) => {
  const resp = await api.get('api/v1/message-favorite-tags', { params: scopeParams(scope, worldId) })
  return {
    items: Array.isArray(resp.data?.items) ? resp.data.items.map(normalizeTag) as MessageFavoriteTag[] : [],
    canManageWorld: resp.data?.canManageWorld === true,
  }
}

export const createMessageFavoriteTag = async (scope: MessageFavoriteScope, worldId: string, name: string) => {
  const resp = await api.post('api/v1/message-favorite-tags', { ...scopeParams(scope, worldId), name })
  return normalizeTag(resp.data?.item)
}

export const renameMessageFavoriteTag = async (id: string, name: string) => {
  const resp = await api.patch(`api/v1/message-favorite-tags/${encodeURIComponent(id)}`, { name })
  return normalizeTag(resp.data?.item)
}

export const deleteMessageFavoriteTag = async (id: string) => {
  await api.delete(`api/v1/message-favorite-tags/${encodeURIComponent(id)}`)
}
