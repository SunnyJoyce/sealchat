<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { NButton, NEmpty, NIcon, NInput, NSelect, NSpin, NTag, useMessage, useThemeVars } from 'naive-ui'
import { Bookmark, GripVertical, Minus, Search, X } from '@vicons/tabler'
import { chatEvent, useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { findChannelByIdFromTree } from '@/stores/chatChannelSelection'
import { navigateToMessageTarget } from '@/utils/messageJump'
import { describeMessageFavoriteError, type MessageFavoriteItem, type MessageFavoriteListQuery, type MessageFavoriteScope } from './api'
import { messageFavoriteBucketKey, useMessageFavoriteStore } from './store'
import { useFavoriteWindowLayout, type FavoriteResizeDirection } from './useFavoriteWindowLayout'
import { useFavoritePointerSort } from './useFavoritePointerSort'
import MessageFavoriteCard from './MessageFavoriteCard.vue'
import MessageFavoriteTagManager from './MessageFavoriteTagManager.vue'

// 唯一的消息收藏悬浮窗。由聊天壳层挂载一次；入口只调用 open()。
// 领域数据来自 store；位置、尺寸、最小化和筛选输入是本窗口的 UI 状态。

const props = withDefaults(defineProps<{ available?: boolean, embedded?: boolean }>(), { available: true, embedded: false })

const WORLD_CHANGED_EVENT = 'message.favorite.world.changed'
const RESIZE_DIRECTIONS: FavoriteResizeDirection[] = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']

const chat = useChatStore()
const user = useUserStore()
const store = useMessageFavoriteStore()
const notice = useMessage()
const themeVars = useThemeVars()

const FILTER_STORAGE_PREFIX = 'sealchat:message-favorites:filters:v1'
const userId = computed(() => String(user.info?.id || ''))
const currentWorldId = computed(() => String(chat.currentWorldId || '').trim())
const win = useFavoriteWindowLayout(userId, { embedded: props.embedded })
const listRef = ref<HTMLElement | null>(null)
const attention = ref(false)
const jumpingId = ref('')
let attentionTimer: ReturnType<typeof setTimeout> | undefined
let keywordTimer: ReturnType<typeof setTimeout> | undefined

const createFilter = () => ({ keyword: '', q: '', tagIds: [] as string[], worldId: '' })
const scope = ref<MessageFavoriteScope>(currentWorldId.value ? 'world' : 'personal')
const filters = reactive({
  personal: createFilter(),
  world: createFilter(),
})
const filterStorageKey = computed(() => `${FILTER_STORAGE_PREFIX}:${userId.value}`)

const normalizeStoredFilter = (value: unknown) => {
  const raw = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  const keyword = typeof raw.keyword === 'string' ? raw.keyword : ''
  return {
    keyword,
    q: keyword.trim(),
    tagIds: Array.isArray(raw.tagIds) ? raw.tagIds.map((id) => String(id || '').trim()).filter(Boolean).slice(0, 100) : [],
    worldId: typeof raw.worldId === 'string' ? raw.worldId : '',
  }
}

const readFilterMemory = () => {
  if (!userId.value) return null
  try {
    const parsed = JSON.parse(localStorage.getItem(filterStorageKey.value) || 'null') as unknown
    if (!parsed || typeof parsed !== 'object' || (parsed as Record<string, unknown>).version !== 1) return null
    return parsed as { version: 1, personal?: unknown, worlds?: Record<string, unknown> }
  } catch {
    return null
  }
}

const loadFilterMemory = () => {
  const saved = readFilterMemory()
  Object.assign(filters.personal, normalizeStoredFilter(saved?.personal))
  const worldSaved = currentWorldId.value ? saved?.worlds?.[currentWorldId.value] : null
  Object.assign(filters.world, normalizeStoredFilter(worldSaved))
  filters.world.worldId = ''
}

const persistFilterMemory = () => {
  if (!userId.value) return
  const saved = readFilterMemory()
  const worlds = saved?.worlds && typeof saved.worlds === 'object' ? { ...saved.worlds } : {}
  if (currentWorldId.value) {
    worlds[currentWorldId.value] = {
      keyword: filters.world.keyword,
      tagIds: [...filters.world.tagIds],
    }
  }
  try {
    localStorage.setItem(filterStorageKey.value, JSON.stringify({
      version: 1,
      personal: {
        keyword: filters.personal.keyword,
        tagIds: [...filters.personal.tagIds],
        worldId: filters.personal.worldId,
      },
      worlds,
    }))
  } catch {
    // 存储不可用时只失去筛选记忆。
  }
}
const worldName = (worldId: string) => String((chat.worldMap as Record<string, any>)?.[worldId]?.name || '').trim()
const currentWorldName = computed(() => worldName(currentWorldId.value) || '当前世界')
const active = computed(() => filters[scope.value])
const bucketWorldId = computed(() => (scope.value === 'world' ? currentWorldId.value : ''))
const bucketKey = computed(() => messageFavoriteBucketKey(scope.value, bucketWorldId.value))
const list = computed(() => store.listState(bucketKey.value))
const items = computed(() => list.value?.items || [])
const tags = computed(() => store.tagItems(scope.value, bucketWorldId.value))
const visible = computed(() => props.available && win.layout.mode !== 'hidden')
const expanded = computed(() => props.available && win.layout.mode === 'expanded')
const canMaintain = computed(() => scope.value === 'personal' || store.canManageWorld(currentWorldId.value) === true)
const readOnly = computed(() => scope.value === 'world' && store.canManageWorld(currentWorldId.value) === false)
const filtersActive = computed(() => !!active.value.q || active.value.tagIds.length > 0 || (scope.value === 'personal' && !!filters.personal.worldId))
const sortable = computed(() => canMaintain.value && !filtersActive.value && !list.value?.nextCursor && !list.value?.loading && !list.value?.moving && items.value.length > 1)
const sortHint = computed(() => {
  if (filtersActive.value) return '清除筛选后可调整排序'
  if (list.value?.nextCursor) return '加载全部收藏后可调整排序'
  if (list.value?.moving) return '正在保存排序'
  return '拖动排序，也可用方向键上下移动'
})

const query = computed<MessageFavoriteListQuery>(() => ({
  scope: scope.value,
  worldId: scope.value === 'world' ? currentWorldId.value : filters.personal.worldId,
  q: active.value.q,
  tagIds: [...active.value.tagIds],
}))
const querySignature = computed(() => JSON.stringify(query.value))

const worldOptions = computed(() => {
  const options = new Map<string, string>()
  for (const option of chat.joinedWorldOptions as Array<{ value: string, label: string }>) options.set(option.value, option.label)
  for (const item of items.value) {
    if (item.worldId && !options.has(item.worldId)) options.set(item.worldId, worldName(item.worldId) || '未知世界')
  }
  return Array.from(options, ([value, label]) => ({ value, label }))
})

const sourceLabel = (item: MessageFavoriteItem) => {
  const channel = findChannelByIdFromTree((chat.channelTreeByWorld as Record<string, any[]>)?.[item.worldId] || [], item.channelId) as { name?: string } | null
  const channelLabel = channel?.name ? `#${channel.name}` : '#频道'
  if (scope.value === 'world') return channelLabel
  return `${worldName(item.worldId) || '未知世界'} · ${channelLabel}`
}

const emptyText = computed(() => {
  if (filtersActive.value) return '没有符合筛选条件的收藏'
  if (scope.value === 'world' && readOnly.value) return '这个世界还没有世界收藏'
  return '还没有收藏。可在消息右键菜单或多选工具栏中收藏消息。'
})

const sort = useFavoritePointerSort({
  scroller: listRef,
  enabled: () => sortable.value,
  onCommit: (id, toIndex) => void moveTo(id, toIndex),
})

const moveTo = async (id: string, toIndex: number, refocus = false) => {
  try {
    const moved = await store.moveFavorite(bucketKey.value, id, toIndex)
    if (moved && refocus) {
      await nextTick()
      listRef.value?.querySelector<HTMLElement>(`[data-favorite-sort-id="${CSS.escape(id)}"] .mf-handle`)?.focus()
    }
  } catch (error) {
    notice.error(`排序失败，已恢复服务端顺序：${describeMessageFavoriteError(error, '请稍后重试')}`)
  }
}

const moveActionsFor = (index: number) => (sortable.value
  ? { top: index > 0, up: index > 0, down: index < items.value.length - 1 }
  : null)

const handleMove = (item: MessageFavoriteItem, index: number, action: 'up' | 'down' | 'top') => {
  const toIndex = action === 'top' ? 0 : action === 'up' ? index - 1 : index + 1
  void moveTo(item.id, toIndex)
}

const handleHandleKey = (event: KeyboardEvent, index: number, item: MessageFavoriteItem) => {
  if (!sortable.value) return
  if (event.key === 'ArrowUp' && index > 0) {
    event.preventDefault()
    void moveTo(item.id, index - 1, true)
  } else if (event.key === 'ArrowDown' && index < items.value.length - 1) {
    event.preventDefault()
    void moveTo(item.id, index + 1, true)
  }
}

const reload = () => {
  if (!expanded.value) return
  if (scope.value === 'world' && !currentWorldId.value) return
  void store.loadList(query.value)
  void store.ensureTags(scope.value, bucketWorldId.value).catch(() => undefined)
}

const applyKeyword = () => {
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
  keywordTimer = undefined
  active.value.q = active.value.keyword.trim()
}

const updateKeyword = (value: string) => {
  active.value.keyword = value
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(applyKeyword, 300)
}

const toggleTagFilter = (tagId: string) => {
  const ids = active.value.tagIds
  const index = ids.indexOf(tagId)
  if (index >= 0) ids.splice(index, 1)
  else ids.push(tagId)
}

const clearFilters = () => {
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
  keywordTimer = undefined
  Object.assign(active.value, { keyword: '', q: '', tagIds: [], worldId: '' })
}

const selectScope = (next: MessageFavoriteScope) => {
  if (next === 'world' && !currentWorldId.value) return
  scope.value = next
}

const jump = async (item: MessageFavoriteItem) => {
  if (!item.message || jumpingId.value) return
  jumpingId.value = item.id
  try {
    const jumped = await navigateToMessageTarget(chat, {
      worldId: item.worldId,
      channelId: item.channelId,
      messageId: item.messageId,
      createdAt: (item.message as any).createdAt,
      displayOrder: (item.message as any).displayOrder,
    })
    // 小屏上窗口会盖住消息，跳转成功后收起为悬浮图标。
    if (jumped && win.compact.value) win.setMode('minimized')
  } catch (error) {
    notice.error(describeMessageFavoriteError(error, '无法定位原消息'))
  } finally {
    jumpingId.value = ''
  }
}

const remove = async (item: MessageFavoriteItem) => {
  try {
    await store.removeFavorite(item.id)
    notice.success('已取消收藏')
  } catch (error) {
    notice.error(describeMessageFavoriteError(error, '取消收藏失败'))
  }
}

const handleListScroll = () => {
  const element = listRef.value
  if (!element || !list.value?.nextCursor) return
  if (element.scrollTop + element.clientHeight >= element.scrollHeight - 160) void store.loadMore(bucketKey.value)
}

const handleWorldChanged = (event: any) => {
  const payload = event?.argv?.options || event?.argv?.Options || {}
  const worldId = String(payload.worldId || '')
  if (!expanded.value || scope.value !== 'world' || worldId !== currentWorldId.value) return
  store.handleWorldChanged(worldId)
}

const handleBadgeEnd = (event: PointerEvent) => {
  if (win.onPointerEnd(event)) restore()
}

const restore = () => win.setMode('expanded')

const open = () => {
  if (!props.available) return
  scope.value = currentWorldId.value ? 'world' : 'personal'
  win.setMode('expanded')
  attention.value = true
  if (attentionTimer !== undefined) clearTimeout(attentionTimer)
  attentionTimer = setTimeout(() => { attention.value = false }, 700)
}

const close = () => {
  sort.cancel()
  win.setMode('hidden')
}

// 重新展开、切换作用域/世界或筛选变化时按当前上下文重读；store 用请求序号丢弃过期响应。
watch([expanded, querySignature], ([isExpanded]) => {
  if (isExpanded) reload()
}, { immediate: true })

watch(userId, () => {
  loadFilterMemory()
  scope.value = currentWorldId.value ? 'world' : 'personal'
}, { immediate: true })

watch(filters, persistFilterMemory, { deep: true })

watch(currentWorldId, (worldId, previous) => {
  if (worldId === previous) return
  sort.cancel()
  const saved = readFilterMemory()
  Object.assign(filters.world, normalizeStoredFilter(worldId ? saved?.worlds?.[worldId] : null))
  filters.world.worldId = ''
  if (!worldId && scope.value === 'world') scope.value = 'personal'
})

// 标签真正加载完成后再清理已不存在的筛选项，避免恢复筛选时被初始空列表误清除。
watch([tags, () => store.tagsLoaded(scope.value, bucketWorldId.value)], ([next, loaded]) => {
  if (!loaded) return
  const ids = new Set(next.map((tag) => tag.id))
  const kept = active.value.tagIds.filter((id) => ids.has(id))
  if (kept.length !== active.value.tagIds.length) active.value.tagIds = kept
})

watch(() => props.available, (available) => {
  if (!available) sort.cancel()
})

onMounted(() => chatEvent.on(WORLD_CHANGED_EVENT as any, handleWorldChanged as any))
onBeforeUnmount(() => {
  chatEvent.off(WORLD_CHANGED_EVENT as any, handleWorldChanged as any)
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
  if (attentionTimer !== undefined) clearTimeout(attentionTimer)
})

const rootStyle = computed(() => ({
  '--mf-accent': `var(--primary-color, ${themeVars.value.primaryColor})`,
}))

defineExpose({ open })
</script>

<template>
  <Teleport to="body">
    <Transition name="mf-window">
      <section
        v-if="visible && win.layout.mode === 'expanded'"
        class="mf-window sc-overlay"
        :class="{ 'is-compact': win.compact.value, 'is-attention': attention, 'is-gesture': !!win.gesture.value }"
        data-sc-ui="floating-panel"
        data-sc-font-surface="true"
        role="dialog"
        aria-label="消息收藏"
        :style="{
          ...rootStyle,
          left: `${win.rect.value.x}px`,
          top: `${win.rect.value.y}px`,
          width: `${win.rect.value.width}px`,
          height: `${win.rect.value.height}px`,
        }"
      >
        <header
          class="mf-window__titlebar"
          @pointerdown="win.startMove"
          @pointermove="win.onPointerMove"
          @pointerup="win.onPointerEnd"
          @pointercancel="win.onPointerEnd"
          @lostpointercapture="win.onPointerEnd"
        >
          <NIcon :size="17" class="mf-window__mark"><Bookmark /></NIcon>
          <span class="mf-window__title">消息收藏</span>
          <span class="mf-window__spacer" />
          <button type="button" class="mf-window__icon-btn" aria-label="最小化" title="最小化" @click="win.setMode('minimized')">
            <NIcon :size="16"><Minus /></NIcon>
          </button>
          <button type="button" class="mf-window__icon-btn" aria-label="关闭消息收藏" title="关闭" @click="close">
            <NIcon :size="16"><X /></NIcon>
          </button>
        </header>

        <nav class="mf-window__scopes" role="tablist" aria-label="收藏作用域">
          <button
            type="button"
            role="tab"
            class="mf-window__scope"
            :class="{ 'is-active': scope === 'world' }"
            :aria-selected="scope === 'world'"
            :disabled="!currentWorldId"
            :title="currentWorldId ? `世界收藏 · ${currentWorldName}` : '进入世界后可查看世界收藏'"
            @click="selectScope('world')"
          >
            <span class="mf-window__scope-text">世界收藏<template v-if="currentWorldId"> · {{ currentWorldName }}</template></span>
          </button>
          <button
            type="button"
            role="tab"
            class="mf-window__scope"
            :class="{ 'is-active': scope === 'personal' }"
            :aria-selected="scope === 'personal'"
            @click="selectScope('personal')"
          >
            个人收藏
          </button>
        </nav>

        <div class="mf-window__toolbar">
          <NInput
            :value="active.keyword"
            size="small"
            round
            clearable
            placeholder="搜索收藏内容"
            class="mf-window__search"
            @update:value="updateKeyword"
            @keydown.enter.prevent="applyKeyword"
          >
            <template #prefix><NIcon><Search /></NIcon></template>
          </NInput>
          <NSelect
            v-if="scope === 'personal'"
            :value="filters.personal.worldId || null"
            size="small"
            clearable
            filterable
            placeholder="全部世界"
            class="mf-window__world-filter"
            :options="worldOptions"
            :consistent-menu-width="false"
            @update:value="(value: string | null) => { filters.personal.worldId = value || '' }"
          />
          <MessageFavoriteTagManager v-if="canMaintain" :scope="scope" :world-id="bucketWorldId" :tags="tags" />
        </div>

        <div v-if="tags.length" class="mf-window__tags" role="group" aria-label="按标签筛选（同时满足所选标签）">
          <NTag
            v-for="tag in tags"
            :key="tag.id"
            size="small"
            round
            checkable
            :checked="active.tagIds.includes(tag.id)"
            @update:checked="toggleTagFilter(tag.id)"
          >
            {{ tag.name }}
          </NTag>
        </div>

        <div v-if="readOnly || filtersActive" class="mf-window__status" role="status">
          <span v-if="readOnly">只读 · 世界收藏由世界管理员维护</span>
          <span v-else-if="filtersActive && canMaintain && items.length > 1">筛选中 · 清除筛选后可调整排序</span>
          <span v-else-if="filtersActive">筛选中</span>
          <button v-if="filtersActive" type="button" class="mf-window__link" @click="clearFilters">清除筛选</button>
        </div>

        <div ref="listRef" class="mf-window__list" @scroll.passive="handleListScroll">
          <div v-if="list?.error && !items.length" class="mf-window__empty">
            <NEmpty size="small" :description="list.error">
              <template #extra><NButton size="small" @click="reload">重试</NButton></template>
            </NEmpty>
          </div>
          <div v-else-if="!items.length && (!list?.loaded || list?.loading)" class="mf-window__empty">
            <NSpin size="small" />
          </div>
          <div v-else-if="!items.length" class="mf-window__empty">
            <NEmpty size="small" :description="emptyText" />
          </div>
          <template v-else>
            <MessageFavoriteCard
              v-for="(item, index) in items"
              :key="item.id"
              :item="item"
              :tags="tags"
              :current-user-id="userId"
              :source-label="sourceLabel(item)"
              :can-maintain="canMaintain"
              :move-actions="moveActionsFor(index)"
              :active-tag-ids="active.tagIds"
              :sorting="sort.sortingId.value === item.id"
              :drop-before="!!sort.sortingId.value && sort.dropIndex.value === index"
              :jumping="jumpingId === item.id"
              @jump="jump(item)"
              @remove="remove(item)"
              @move="handleMove(item, index, $event)"
              @filter-tag="toggleTagFilter"
            >
              <template v-if="canMaintain" #handle>
                <button
                  type="button"
                  class="mf-handle"
                  :disabled="!sortable"
                  :aria-label="sortable ? '拖动排序，可用方向键上下移动' : sortHint"
                  :title="sortHint"
                  @pointerdown="sort.begin($event, item.id)"
                  @pointermove="sort.move"
                  @pointerup="sort.end"
                  @pointercancel="sort.end"
                  @lostpointercapture="sort.end"
                  @keydown="handleHandleKey($event, index, item)"
                >
                  <NIcon :size="16"><GripVertical /></NIcon>
                </button>
              </template>
            </MessageFavoriteCard>
            <div
              class="mf-window__end-drop"
              :class="{ 'is-active': !!sort.sortingId.value && sort.dropIndex.value === items.length }"
            />
            <div v-if="list?.loadingMore" class="mf-window__more"><NSpin size="small" /></div>
            <button v-else-if="list?.nextCursor" type="button" class="mf-window__more mf-window__link" @click="store.loadMore(bucketKey)">
              加载更多
            </button>
          </template>
        </div>

        <div
          v-for="direction in RESIZE_DIRECTIONS"
          :key="direction"
          class="mf-window__resize"
          :class="`is-${direction}`"
          aria-hidden="true"
          @pointerdown="win.startResize(direction, $event)"
          @pointermove="win.onPointerMove"
          @pointerup="win.onPointerEnd"
          @pointercancel="win.onPointerEnd"
          @lostpointercapture="win.onPointerEnd"
        />
      </section>
    </Transition>

    <button
      v-if="visible && win.layout.mode === 'minimized'"
      type="button"
      class="mf-badge sc-overlay"
      data-sc-ui="floating-panel"
      aria-label="恢复消息收藏窗口"
      title="消息收藏（点击恢复，可拖动）"
      :style="{ ...rootStyle, left: `${win.badge.value.x}px`, top: `${win.badge.value.y}px` }"
      @pointerdown="win.startBadge"
      @pointermove="win.onPointerMove"
      @pointerup="handleBadgeEnd"
      @pointercancel="win.onPointerEnd"
      @lostpointercapture="win.onPointerEnd"
      @click="(event: MouseEvent) => { if (event.detail === 0) restore() }"
    >
      <NIcon :size="22"><Bookmark /></NIcon>
    </button>
  </Teleport>
</template>

<style scoped>
.mf-window {
  position: fixed;
  z-index: 1210;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;
  border: 1px solid;
  border-radius: var(--sc-overlay-radius, 12px);
  background: var(--sc-overlay-background, var(--sc-bg-elevated));
  box-shadow: var(--sc-overlay-shadow, var(--chat-message-shadow));
  transition: outline-color 0.3s ease;
  outline: 2px solid transparent;
  outline-offset: 2px;
}

.mf-window.is-attention {
  outline-color: var(--mf-accent);
}

.mf-window.is-gesture {
  user-select: none;
}

.mf-window__titlebar {
  display: flex;
  flex: none;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 6px 0 12px;
  cursor: move;
  touch-action: none;
  user-select: none;
}

.mf-window__mark {
  color: var(--mf-accent);
}

.mf-window__title {
  font-size: 14px;
  font-weight: 600;
}

.mf-window__spacer {
  flex: 1;
}

.mf-window__icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--sc-text-secondary);
  cursor: pointer;
}

.mf-window__icon-btn:hover {
  background: var(--sc-bg-hover);
  color: var(--sc-text-primary);
}

.mf-window__scopes {
  display: flex;
  flex: none;
  gap: 2px;
  margin: 0 12px;
  padding: 2px;
  border-radius: 8px;
  background: var(--sc-bg-layer);
}

.mf-window__scope {
  overflow: hidden;
  flex: 1;
  min-width: 0;
  min-height: 30px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--sc-text-secondary);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.mf-window__scope-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mf-window__scope.is-active {
  background: var(--sc-bg-elevated);
  box-shadow: inset 0 0 0 1px var(--sc-border-mute);
  color: var(--sc-text-primary);
  font-weight: 600;
}

.mf-window__scope:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.mf-window__toolbar {
  display: flex;
  flex: none;
  align-items: center;
  gap: 6px;
  padding: 10px 12px 6px;
}

.mf-window__search {
  flex: 1;
  min-width: 0;
}

.mf-window__world-filter {
  flex: 0 1 128px;
  min-width: 96px;
}

.mf-window__tags {
  display: flex;
  flex: none;
  gap: 6px;
  overflow-x: auto;
  padding: 2px 12px 6px;
  scrollbar-width: thin;
  touch-action: pan-x;
}

.mf-window__tags :deep(.n-tag) {
  flex: none;
}

.mf-window__status {
  display: flex;
  flex: none;
  align-items: center;
  gap: 8px;
  padding: 0 12px 6px;
  color: var(--sc-text-secondary);
  font-size: 12px;
}

.mf-window__link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--mf-accent);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.mf-window__list {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 10px 12px 12px;
  border-top: 1px solid var(--sc-border-mute);
  scrollbar-width: thin;
  scrollbar-color: var(--sc-scrollbar-thumb) transparent;
}

.mf-window__empty {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 24px 12px;
  text-align: center;
}

.mf-window__end-drop {
  flex: none;
  height: 2px;
  margin: -4px 8px 0;
  border-radius: 2px;
}

.mf-window__end-drop.is-active {
  background: var(--mf-accent);
}

.mf-window__more {
  align-self: center;
  padding: 4px 0;
}

.mf-handle {
  display: inline-flex;
  flex: none;
  align-self: stretch;
  align-items: center;
  justify-content: center;
  width: 24px;
  margin: -4px 0 -4px -6px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--sc-text-secondary);
  cursor: grab;
  touch-action: none;
}

.mf-handle:hover:not(:disabled),
.mf-handle:focus-visible {
  background: var(--sc-bg-hover);
  color: var(--sc-text-primary);
}

.mf-handle:active:not(:disabled) {
  cursor: grabbing;
}

.mf-handle:disabled {
  cursor: not-allowed;
  opacity: 0.35;
}

.mf-window__resize {
  position: absolute;
  z-index: 2;
  touch-action: none;
}

.mf-window__resize.is-n,
.mf-window__resize.is-s { left: 14px; right: 14px; height: 8px; cursor: ns-resize; }
.mf-window__resize.is-n { top: 0; }
.mf-window__resize.is-s { bottom: 0; }
.mf-window__resize.is-e,
.mf-window__resize.is-w { top: 14px; bottom: 14px; width: 8px; cursor: ew-resize; }
.mf-window__resize.is-e { right: 0; }
.mf-window__resize.is-w { left: 0; }
.mf-window__resize.is-ne,
.mf-window__resize.is-nw,
.mf-window__resize.is-se,
.mf-window__resize.is-sw { width: 16px; height: 16px; }
.mf-window__resize.is-ne { top: 0; right: 0; cursor: nesw-resize; }
.mf-window__resize.is-nw { top: 0; left: 0; cursor: nwse-resize; }
.mf-window__resize.is-se { right: 0; bottom: 0; cursor: nwse-resize; }
.mf-window__resize.is-sw { bottom: 0; left: 0; cursor: nesw-resize; }

/* 触摸设备上加大右下角缩放热区，避免与列表滚动争抢。 */
@media (pointer: coarse) {
  .mf-window__resize.is-se { width: 28px; height: 28px; }
  .mf-window__resize.is-n,
  .mf-window__resize.is-s { height: 10px; }
}

.mf-badge {
  position: fixed;
  z-index: 1210;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  padding: 0;
  border: 1px solid;
  border-radius: 50%;
  background: var(--sc-overlay-background, var(--sc-bg-elevated));
  box-shadow: var(--sc-overlay-shadow, var(--chat-message-shadow));
  color: var(--mf-accent);
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.mf-badge:active {
  cursor: grabbing;
}

.mf-window-enter-active,
.mf-window-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.mf-window-enter-from,
.mf-window-leave-to {
  opacity: 0;
  transform: scale(0.98) translateY(6px);
}

@media (prefers-reduced-motion: reduce) {
  .mf-window-enter-active,
  .mf-window-leave-active,
  .mf-window,
  .mf-card {
    transition: none;
  }
}
</style>
