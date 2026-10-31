<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { NCheckbox, NDropdown, NIcon, NTag, type DropdownOption } from 'naive-ui'
import { ArrowUpRight, DotsVertical, Lock, MessageOff } from '@vicons/tabler'
import type { MessageFavoriteItem, MessageFavoriteTag } from './api'
import { buildFavoriteCardView, canJumpToFavorite } from './presentation'
import MessageFavoriteTagPicker from './MessageFavoriteTagPicker.vue'

// 收藏列表的窄窗口消息卡片。只做受控预览，不复用带回复/编辑/reaction 的 chat-item。

const props = defineProps<{
  item: MessageFavoriteItem
  tags: MessageFavoriteTag[]
  currentUserId: string
  sourceLabel: string
  canMaintain: boolean
  // 当前可用的手动排序动作；null 表示不提供菜单排序项。
  moveActions: { up: boolean, down: boolean, top: boolean } | null
  activeTagIds: string[]
  sorting?: boolean
  dropBefore?: boolean
  jumping?: boolean
  // 批量选择模式：用复选框替换排序把手，隐藏单项维护入口，点击卡片切换选择。
  selecting?: boolean
  selected?: boolean
}>()

const emit = defineEmits<{
  (e: 'jump'): void
  (e: 'remove'): void
  (e: 'move', action: 'up' | 'down' | 'top'): void
  (e: 'filter-tag', tagId: string): void
  (e: 'toggle-select'): void
}>()

const view = computed(() => (props.item.message ? buildFavoriteCardView(props.item.message, props.currentUserId) : null))
const jumpable = computed(() => canJumpToFavorite(props.item))
const contentRef = ref<HTMLElement | null>(null)
const expanded = ref(false)
const overflowing = ref(false)

const measure = async () => {
  await nextTick()
  const element = contentRef.value
  overflowing.value = !!element && element.scrollHeight > element.clientHeight + 2
}

onMounted(measure)
watch(() => view.value?.contentHtml, () => {
  expanded.value = false
  void measure()
})

const menuOptions = computed<DropdownOption[]>(() => {
  const options: DropdownOption[] = []
  if (props.selecting) return options
  if (props.moveActions) {
    options.push(
      { key: 'top', label: '移到顶部', disabled: !props.moveActions.top },
      { key: 'up', label: '上移一位', disabled: !props.moveActions.up },
      { key: 'down', label: '下移一位', disabled: !props.moveActions.down },
      { type: 'divider', key: 'divider' },
    )
  }
  if (props.canMaintain) options.push({ key: 'remove', label: '取消收藏' })
  return options
})

const handleMenu = (key: string) => {
  if (key === 'remove') emit('remove')
  else if (key === 'up' || key === 'down' || key === 'top') emit('move', key)
}

// 预览中的链接不在当前聊天页内跳走。
const handleContentClick = (event: MouseEvent) => {
  const anchor = (event.target as HTMLElement | null)?.closest('a[href]') as HTMLAnchorElement | null
  if (!anchor) return
  event.preventDefault()
  const href = anchor.href
  if (/^https?:/i.test(href)) window.open(href, '_blank', 'noopener,noreferrer')
}

// 选择模式下点击卡片空白处切换选择；链接、按钮、复选框和正在进行的文字选择保持原行为。
const handleCardClick = (event: MouseEvent) => {
  if (!props.selecting) return
  const target = event.target as HTMLElement | null
  if (target?.closest('a, button, input, label, .n-checkbox')) return
  if (window.getSelection()?.toString()) return
  emit('toggle-select')
}

const tombstoneText = computed(() => (props.item.status === 'deleted' ? '原消息已删除' : '当前无权查看原消息'))
</script>

<template>
  <article
    class="mf-card"
    :class="{
      'is-tombstone': item.status !== 'available',
      'is-sorting': sorting,
      'is-drop-before': dropBefore,
      'is-selectable': selecting,
      'is-selected': selecting && selected,
    }"
    :data-favorite-sort-id="item.id"
    @click="handleCardClick"
  >
    <div v-if="selecting" class="mf-card__select">
      <NCheckbox
        :checked="selected"
        :aria-label="selected ? '取消选择这条收藏' : '选择这条收藏'"
        @update:checked="emit('toggle-select')"
      />
    </div>
    <slot v-else name="handle" />
    <div class="mf-card__main">
      <template v-if="view">
        <header class="mf-card__meta">
          <img class="mf-card__avatar" :src="view.avatarUrl" alt="" loading="lazy" draggable="false">
          <span class="mf-card__name" :style="view.senderColor ? { color: view.senderColor } : undefined">{{ view.senderName }}</span>
          <span class="mf-card__time" :title="view.timeTitle">{{ view.timeLabel }}</span>
          <NTag v-if="view.isWhisper" size="tiny" :bordered="false" round>
            <template #icon><NIcon><Lock /></NIcon></template>悄悄话
          </NTag>
          <NTag v-else-if="view.isOoc" size="tiny" :bordered="false" round>场外</NTag>
          <span class="mf-card__spacer" />
          <button
            v-if="jumpable"
            type="button"
            class="mf-icon-btn"
            :disabled="jumping"
            aria-label="跳转到原消息"
            title="跳转到原消息"
            @click="emit('jump')"
          >
            <NIcon :size="16"><ArrowUpRight /></NIcon>
          </button>
          <NDropdown v-if="menuOptions.length" trigger="click" placement="bottom-end" :options="menuOptions" @select="handleMenu">
            <button type="button" class="mf-icon-btn" aria-label="更多操作" title="更多操作">
              <NIcon :size="16"><DotsVertical /></NIcon>
            </button>
          </NDropdown>
        </header>
        <div class="mf-card__source" :title="sourceLabel">{{ sourceLabel }}</div>
        <div
          ref="contentRef"
          class="mf-card__content"
          :class="{ 'is-expanded': expanded }"
          @click="handleContentClick"
          v-html="view.contentHtml"
        />
        <button
          v-if="overflowing || expanded"
          type="button"
          class="mf-card__expand"
          @click="expanded = !expanded"
        >
          {{ expanded ? '收起' : '展开全文' }}
        </button>
      </template>
      <template v-else>
        <header class="mf-card__meta">
          <NIcon :size="16" class="mf-card__tomb-icon"><MessageOff /></NIcon>
          <span class="mf-card__tomb-text">{{ tombstoneText }}</span>
          <span class="mf-card__spacer" />
          <NDropdown v-if="menuOptions.length" trigger="click" placement="bottom-end" :options="menuOptions" @select="handleMenu">
            <button type="button" class="mf-icon-btn" aria-label="更多操作" title="更多操作">
              <NIcon :size="16"><DotsVertical /></NIcon>
            </button>
          </NDropdown>
        </header>
        <div v-if="sourceLabel" class="mf-card__source">{{ sourceLabel }}</div>
      </template>
      <footer v-if="item.tags.length || (canMaintain && !selecting)" class="mf-card__tags">
        <NTag
          v-for="tag in item.tags"
          :key="tag.id"
          size="small"
          round
          :checkable="!selecting"
          :checked="activeTagIds.includes(tag.id)"
          :title="selecting ? tag.name : activeTagIds.includes(tag.id) ? `取消筛选「${tag.name}」` : `按「${tag.name}」筛选`"
          @update:checked="emit('filter-tag', tag.id)"
        >
          {{ tag.name }}
        </NTag>
        <MessageFavoriteTagPicker v-if="canMaintain && !selecting" :item="item" :tags="tags" :can-maintain="canMaintain" />
      </footer>
    </div>
  </article>
</template>

<style scoped>
.mf-card {
  position: relative;
  display: flex;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--sc-border-mute);
  border-radius: 10px;
  background: var(--sc-bg-layer);
  transition: opacity 0.15s ease, background-color 0.15s ease;
}

.mf-card.is-tombstone {
  background: transparent;
  border-style: dashed;
  color: var(--sc-text-secondary);
}

.mf-card.is-selectable {
  cursor: pointer;
}

.mf-card.is-selected {
  border-color: var(--mf-accent);
  border-style: solid;
  box-shadow: inset 0 0 0 1px var(--mf-accent);
}

@supports (color: color-mix(in srgb, red, transparent)) {
  .mf-card.is-selected {
    background: color-mix(in srgb, var(--mf-accent) 8%, var(--sc-bg-layer));
  }
}

.mf-card__select {
  display: flex;
  flex: none;
  align-items: flex-start;
  justify-content: center;
  width: 24px;
  margin: 0 0 0 -4px;
  padding-top: 3px;
}

.mf-card.is-sorting {
  opacity: 0.45;
}

.mf-card.is-drop-before::before {
  position: absolute;
  top: -6px;
  left: 8px;
  right: 8px;
  height: 2px;
  border-radius: 2px;
  background: var(--mf-accent);
  content: '';
  pointer-events: none;
}

.mf-card__main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.mf-card__meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  min-height: 24px;
}

.mf-card__avatar {
  flex: none;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  object-fit: cover;
}

.mf-card__name {
  overflow: hidden;
  min-width: 2.5em;
  color: var(--sc-text-primary);
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mf-card__time,
.mf-card__source {
  overflow: hidden;
  flex: 0 1 auto;
  min-width: 0;
  color: var(--sc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mf-card__spacer {
  flex: 1;
}

.mf-card__tomb-icon {
  color: var(--sc-text-secondary);
}

.mf-card__tomb-text {
  font-size: 13px;
}

.mf-card__content {
  overflow: hidden;
  max-height: 9.6em;
  color: var(--sc-text-primary);
  font-size: 13px;
  line-height: 1.6;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.mf-card__content.is-expanded {
  max-height: none;
}

.mf-card__content :deep(p) {
  margin: 0;
}

.mf-card__content :deep(img) {
  display: block;
  max-width: 100%;
  max-height: 160px;
  margin: 4px 0;
  border-radius: 6px;
  object-fit: contain;
}

.mf-card__content :deep(a) {
  color: var(--mf-accent);
}

.mf-card__expand {
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--mf-accent);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.mf-card__tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
}

.mf-icon-btn {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--sc-text-secondary);
  cursor: pointer;
}

.mf-icon-btn:hover:not(:disabled) {
  background: var(--sc-bg-hover);
  color: var(--sc-text-primary);
}

.mf-icon-btn:disabled {
  cursor: default;
  opacity: 0.5;
}
</style>
