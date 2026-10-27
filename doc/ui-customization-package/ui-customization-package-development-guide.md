# SealChat UI 自定义包开发指南

## 1. 适用范围

本文面向想为 SealChat 制作界面主题、世界皮肤和高级 CSS 包的开发者。

SealChat 目前有两套外观机制。传统“颜色主题”只负责一组预定义颜色字段。UI 自定义包可以继续覆盖 CSS 语义变量，也可以按页面和稳定 DOM hook 调整布局、边框、阴影、动画和组件外观。

如果你只想修改主色、聊天气泡颜色和输入框颜色，优先使用传统颜色主题。如果你需要修改登录页、聊天主界面、世界控制台、LOG、OB、Embed 或小剧场的整体视觉，应使用 UI 自定义包。

本文以 V1 运行时为准。低层接口和 JavaScript Runtime 说明见 `doc/ui-customization-css-api.md`。架构设计见 `docs/ui-customization-package-design.md`。

## 2. 快速开始

仓库提供一个完整的赛博朋克参考包：

- 可直接导入：`doc/examples/sealchat-ui-package-cyberpunk-neon.json`。
- 可读 CSS 源码：`doc/examples/cyberpunk-ui-package.css`。

导入后，SealChat 会创建一个新的停用包。先检查 CSS，再手动启用。示例包不包含 JavaScript。

如果 CSS 导致页面无法操作，在当前地址增加 `safe-ui=1` 并重新加载。例如：

~~~text
/?safe-ui=1#/worlds
/#/WORLD/CHANNEL?safe-ui=1
~~~

也可以在“显示设置 → 外观 → 界面自定义安全模式”中临时停用当前标签页的全部 UI 自定义。

## 3. UI Package 文件格式

单包使用 `sealchat.ui-package` 格式。

~~~json
{
  "format": "sealchat.ui-package",
  "formatVersion": 1,
  "runtimeApiVersion": 1,
  "manifest": {
    "name": "示例主题",
    "description": "主题说明"
  },
  "targets": ["chat"],
  "css": "...",
  "js": ""
}
~~~

字段含义如下：

| 字段 | 说明 |
| --- | --- |
| `format` | 固定为 `sealchat.ui-package`。 |
| `formatVersion` | 当前为 `1`。 |
| `runtimeApiVersion` | 当前为 `1`。纯 CSS 包也保留这个字段。 |
| `manifest.name` | 导入后的包名称。最长 100 字符。 |
| `manifest.description` | 包说明。最长 500 字符。 |
| `targets` | 包生效的 Surface。至少 1 项。 |
| `css` | CSS 源码。单包上限 256 KiB。 |
| `js` | JavaScript 源码。纯主题包应留空。 |

CSS 和 JavaScript 合计最多 384 KiB。一个平台或世界最多保存 50 个包。

导入文件不携带数据库 ID、worldId、scope、审计字段、脚本 hash 和用户信任状态。导入目标由当前管理入口决定。平台管理中导入会创建平台包，世界管理中导入会创建当前世界的包。

## 4. Surface

Surface 是 UI 包选择页面的稳定接口。不要根据 URL 文本自己猜页面类型。

当前 Surface 如下：

| Surface | 页面 |
| --- | --- |
| `auth` | 登录、注册、找回密码和重置密码。 |
| `lobby` | 世界大厅、关于页等无世界上下文页面。 |
| `world-detail` | 世界详情和世界控制台。 |
| `chat` | 根 `/` 与世界频道聊天主界面。 |
| `split` | 分屏界面。 |
| `embed` | 频道嵌入页。公共 Surface。 |
| `theater` | World Theater。 |
| `observer` | OB 旁观页。公共 Surface。 |
| `invite` | 邀请入口。 |
| `log-layout` | LOG 排版编辑页。 |
| `log-public` | LOG 公开发布页。公共 Surface。 |
| `status` | 状态页。 |
| `internal` | Internal Surface。V1 永远不加载 UI Package。 |
| `global` | 匹配所有普通 Surface 的目标，不是一个真实页面。 |

`global` 当前覆盖 `auth`、`lobby`、`world-detail`、`chat`、`split`、`theater`、`invite`、`log-layout` 和 `status`。

公共 Surface 不会被 `global` 命中。要覆盖 OB、Embed 或公开 LOG，必须把 `observer`、`embed` 和 `log-public` 明确写入 `targets`。

完整参考包建议使用：

~~~json
"targets": ["global", "observer", "embed", "log-public"]
~~~

这样可以覆盖全部可公开定制的 V1 Surface，同时不会进入 `internal`。

## 5. CSS 层叠顺序

SealChat 的外观大致按下面的顺序生效：

~~~text
SealChat 基础样式
< 日间 / 夜间基础配色
< 平台或个人颜色主题
< Platform UI Package
< World UI Package
~~~

同一作用域内，包按管理界面中的顺序加载。靠后的包更容易覆盖前面的包。

世界包只在当前世界生效。平台包可以覆盖登录页和无世界页面。

如果多个包同时修改相同变量，优先调整包顺序。不要为了抢优先级大量使用 `!important`。

## 6. 根节点属性

运行时会在 `html` 上提供稳定属性。它们适合控制页面范围、日夜配色和布局差异。

~~~css
html[data-sc-surface='chat'] {
  /* 仅聊天界面 */
}

html[data-sc-palette='night'] {
  /* 夜间配色 */
}

html[data-sc-layout='compact'] {
  /* 紧凑布局 */
}

html[data-sc-observer='true'] {
  /* 旁观上下文 */
}
~~~

当前公开属性：

| 属性 | 用途 |
| --- | --- |
| `data-sc-surface` | 当前 UI Surface。 |
| `data-sc-palette` | 当前日间或夜间配色。 |
| `data-sc-layout` | 当前显示布局，例如 `compact`。 |
| `data-sc-observer` | 当前是否处于旁观上下文。 |

SealChat 仍保留 `data-display-palette`、`data-display-layout` 和 `data-custom-theme` 等内部主题属性。第三方 UI Package 优先使用 `data-sc-*`。

## 7. 稳定 UI hook

重要区域提供 `data-sc-ui`。这些 hook 比 Vue class、Naive UI 内部 class 和 DOM 层级稳定。

当前可用 hook：

| Hook | 区域 |
| --- | --- |
| `app` | 应用主体。 |
| `header` | 顶部栏。 |
| `sidebar` | 侧边栏。 |
| `channel-list` | 频道列表。 |
| `message-list` | 消息列表。 |
| `message` | 单条消息行，包括置顶消息行。 |
| `composer` | 聊天输入区域。 |
| `world-console` | 新版世界控制台。 |
| `admin-settings` | 平台管理浮层。 |
| `user-profile` | 用户资料浮层。 |
| `ui-customization-trust` | 世界脚本信任提示。 |

推荐写法：

~~~css
[data-sc-surface='chat'] [data-sc-ui='message'] {
  border-inline-start: 2px solid var(--primary-color);
}

[data-sc-ui='header'],
[data-sc-ui='sidebar'] {
  background: var(--sc-bg-surface);
}
~~~

不要把第三方包建立在 `.n-card > .n-card__content > ...` 这类层级上。Naive UI 内部结构不是稳定 API。

## 8. 颜色主题字段和 CSS 变量

传统颜色主题有 21 个字段。UI Package 可以直接设置对应 CSS 变量。

| 颜色主题字段 | CSS 变量 |
| --- | --- |
| `bgSurface` | `--sc-bg-surface` |
| `bgElevated` | `--sc-bg-elevated` |
| `bgInput` | `--sc-bg-input` |
| `bgHeader` | `--sc-bg-header` |
| `textPrimary` | `--sc-text-primary`、聊天内容通常也使用 `--chat-text-primary` |
| `textSecondary` | `--sc-text-secondary`、聊天内容通常也使用 `--chat-text-secondary` |
| `chatIcBg` | `--custom-chat-ic-bg`、`--chat-ic-bg` |
| `chatOocBg` | `--custom-chat-ooc-bg`、`--chat-ooc-bg` |
| `chatStageBg` | `--custom-chat-stage-bg`、`--chat-stage-bg` |
| `chatPreviewBg` | `--custom-chat-preview-bg`、`--chat-preview-bg` |
| `chatPreviewDot` | `--custom-chat-preview-dot`、`--chat-preview-dot` |
| `borderMute` | `--sc-border-mute` |
| `borderStrong` | `--sc-border-strong` |
| `primaryColor` | `--primary-color` |
| `primaryColorHover` | `--primary-color-hover` |
| `actionRibbonHoverText` | `--sc-action-ribbon-hover-text` |
| `keywordBg` | `--custom-keyword-bg` |
| `keywordBorder` | `--custom-keyword-border` |
| `inlineCodeBg` | `--chat-inline-code-bg` |
| `inlineCodeFg` | `--chat-inline-code-fg` |
| `inlineCodeBorder` | `--chat-inline-code-border` |

UI Package 可以覆盖更多语义变量。完整示例包已经演示这些变量。

## 9. 推荐使用的语义变量

### 9.1 页面和组件

~~~css
--sc-bg-page
--sc-bg-surface
--sc-bg-header
--sc-bg-sidebar
--sc-bg-input
--sc-bg-elevated
--sc-bg-layer
--sc-bg-layer-strong
--sc-bg-hover
--sc-bg-active

--sc-text-primary
--sc-text-secondary
--sc-fg-primary
--sc-fg-muted

--sc-border-mute
--sc-border-strong

--primary-color
--primary-color-hover
--sc-action-ribbon-hover-text

--sc-badge-bg
--sc-badge-text
--sc-sidebar-hover
--sc-sidebar-active
--sc-chip-bg

--sc-scrollbar-size
--sc-scrollbar-thumb
--sc-scrollbar-thumb-hover

--sc-font-family
--sc-code-font
~~~

`--sc-font-family` 会影响主界面和多数 Teleport 浮层。不要引用用户本机不存在的单一字体。应提供完整 fallback。

### 9.2 聊天区域

~~~css
--chat-stage-bg
--chat-ic-bg
--chat-ooc-bg
--chat-preview-bg
--chat-preview-dot
--chat-preview-dot-ic
--chat-preview-dot-ooc

--chat-text-primary
--chat-text-secondary
--chat-border-mute
--chat-bubble-border
--chat-ooc-border

--chat-message-shadow
--chat-dice-result-shadow

--chat-archived-bg
--chat-archived-border

--chat-whisper-accent
--chat-whisper-surface
--chat-whisper-bg
--chat-whisper-border
--chat-whisper-tag-bg
--chat-whisper-tag-fg

--chat-inline-code-bg
--chat-inline-code-fg
--chat-inline-code-border

--chat-editing-preview-bg
--chat-editing-preview-ic-bg
--chat-editing-preview-ooc-bg
--chat-live-preview-ic-bg
--chat-live-preview-ooc-bg
~~~

用户设置会动态写入字号、行高、气泡间距、头像尺寸和消息 padding。主题包一般不要强行覆盖这些个人偏好。

### 9.3 音频区域

~~~css
--audio-panel-surface
--audio-panel-border
--audio-panel-shadow
--audio-card-surface
--audio-card-border
--audio-progress-track
--audio-progress-buffer
~~~

### 9.4 玻璃效果

登录背景和部分玻璃组件使用这些变量：

~~~css
--sc-glass-bg
--sc-glass-solid
--sc-glass-border
--sc-glass-shadow
--sc-glass-blur
--sc-glass-saturate
--sc-glass-contrast
~~~

系统还会根据 `--sc-bg-*` 推导页面、卡片、输入框和浮层的玻璃颜色。主题包通常先改语义背景色，再按需要调整玻璃变量。

## 10. 日间和夜间

不要假设用户只使用夜间模式。

推荐先提供共同结构，再按 palette 修改颜色：

~~~css
:root[data-sc-surface] {
  --primary-color: #00d9ff;
}

:root[data-sc-palette='day'] {
  --sc-bg-page: #eafaff;
  --sc-text-primary: #071824;
}

:root[data-sc-palette='night'] {
  --sc-bg-page: #05070d;
  --sc-text-primary: #e8fbff;
}
~~~

如果你的主题只设计了深色版本，可以在说明中写清楚，并同时保证日间模式下文字仍可读。

## 11. 聊天界面

最稳定的修改入口是 `message-list`、`message` 和 `composer`。

~~~css
[data-sc-surface='chat'] [data-sc-ui='message'] {
  border-inline-start: 2px solid color-mix(in srgb, var(--primary-color) 70%, transparent);
}

[data-sc-surface='chat'] [data-sc-ui='composer'] {
  border-top: 1px solid var(--sc-border-strong);
  background: var(--sc-bg-elevated);
}
~~~

需要继续修改消息内部控件时，可以使用现有 class，但要把它视为“兼容性敏感选择器”。例如 `.message-action-bar__btn`、`.typing-preview-bubble`、`.chat-input-actions` 都可能随组件重构变化。

完整参考包把稳定 hook 和兼容性敏感选择器分区书写，便于以后维护。

## 12. Naive UI 组件

Naive UI 会把部分样式写成组件 CSS 变量或内联变量。UI Package 可以覆盖常见组件：

~~~css
:root[data-sc-surface] .n-card {
  --n-color: var(--sc-bg-elevated);
}

:root[data-sc-surface] .n-input {
  --n-color: var(--sc-bg-input);
  --n-text-color: var(--sc-text-primary);
}
~~~

这类选择器不是 SealChat 稳定 hook。升级 Naive UI 后需要重新检查。

优先修改 `--sc-*` 和 `--chat-*`。只有语义变量不能满足效果时，再直接覆盖 `.n-*`。

## 13. Teleport 浮层

下拉菜单、Popover、Dialog、Drawer 和 Notification 可能挂在 `body` 下，不在当前组件 DOM 内。

不要只写：

~~~css
[data-sc-ui='world-console'] .n-popover {
  /* 可能匹配不到 Teleport 后的元素 */
}
~~~

需要统一外观时，可以从根 Surface 限定：

~~~css
:root[data-sc-surface] body .n-popover,
:root[data-sc-surface] body .n-dialog,
:root[data-sc-surface] body .n-drawer {
  color: var(--sc-text-primary);
}
~~~

参考包包含一组 Teleport 浮层样式。

## 14. 公共页面

OB、Embed 和公开 LOG 可能由未登录用户访问。

CSS 可以在平台策略允许时进入这些页面。世界 JavaScript 不进入这些公共上下文。

主题包如果只负责视觉，建议保持 `js` 为空。这样公共页面和普通页面的表现最容易预测。

公开页面还应避免：

- 依赖当前用户资料。
- 依赖世界管理员才有的 DOM。
- 用 `display: none` 隐藏访问控制、错误提示和登录提示。
- 用超大固定尺寸破坏嵌入容器。

## 15. 响应式设计

SealChat 同时运行在桌面浏览器、移动浏览器和嵌入容器中。

至少检查 640 px 以下的窄屏：

~~~css
@media (max-width: 640px) {
  [data-sc-ui='sidebar'] {
    box-shadow: none;
  }

  [data-sc-ui='message'] {
    border-radius: 8px;
  }
}
~~~

不要把侧栏、输入区或弹窗写成固定像素宽度。优先使用 `clamp()`、`min()`、`max()`、百分比和视口单位。

## 16. 动画和可访问性

主题动画应保留 `prefers-reduced-motion`。

~~~css
@media (prefers-reduced-motion: reduce) {
  :root[data-sc-surface] *,
  :root[data-sc-surface] *::before,
  :root[data-sc-surface] *::after {
    animation-duration: 0.001ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.001ms !important;
  }
}
~~~

不要取消键盘焦点。需要重做焦点样式时，至少保留明显的 `:focus-visible` 边框。

文字和背景应保持足够对比度。特别检查次要文字、禁用按钮、代码块、术语高亮和 OOC 气泡。

## 17. 包拆分

一个包可以覆盖所有页面，也可以按用途拆包。

适合拆包的情况包括：

- 平台全局基础配色。
- 单个世界的聊天皮肤。
- 只针对 Theater 的舞台皮肤。
- 只针对 LOG 的发布页皮肤。

不建议把同一套颜色变量拆成很多互相依赖的小包。包顺序会成为额外维护成本。

## 18. 开发和调试

推荐开发流程：

1. 新建一个停用包。
2. 只选择需要测试的 Surface。
3. 先写根变量，再写稳定 `data-sc-ui` hook。
4. 需要时再覆盖兼容性敏感 class。
5. 保存后检查日间和夜间。
6. 检查窄屏。
7. 检查公开页面。
8. 导出 JSON，作为分发文件。

浏览器开发者工具中，可以直接检查 `html` 的 `data-sc-*` 属性和计算后的 CSS 变量。

出现“颜色没有生效”时，按这个顺序排查：

1. Package 是否启用。
2. 当前 Surface 是否在 `targets` 中。
3. 是否有后加载的 Package 覆盖。
4. 元素是否使用了另一个语义变量。
5. 是否被组件内联变量覆盖。
6. 是否需要更具体的选择器。
7. 最后再考虑 `!important`。

## 19. 兼容性规则

第三方包应把接口分成两类。

稳定接口：

- UI Package JSON 格式。
- Surface 名称。
- `data-sc-surface`、`data-sc-palette`、`data-sc-layout` 和 `data-sc-observer`。
- 文档列出的 `data-sc-ui`。
- 核心 `--sc-*` 和 `--chat-*` 语义变量。

兼容性敏感接口：

- Vue 组件 class。
- Naive UI 的 `.n-*` class。
- DOM 层级和子节点顺序。
- 未在本文列出的局部 CSS 变量。
- 带 hash 的构建产物 class 和资源名。

完整参考包会使用少量兼容性敏感选择器来展示效果，但每一组都在 CSS 注释中标出。

## 20. 赛博朋克参考包

`doc/examples/cyberpunk-ui-package.css` 是可读源码。它覆盖：

- 页面、卡片、输入框、浮层和边框语义变量。
- 传统 21 个主题颜色字段对应的全部变量。
- 日间和夜间变体。
- Header、Sidebar、频道列表、消息、输入区、世界控制台、平台管理和用户资料 hook。
- 登录、世界大厅、聊天、分屏、Embed、Theater、OB、邀请、LOG 和状态页 Surface。
- Naive UI 常见卡片、按钮、输入框、选择器、Tab、Drawer、Dialog 和 Popover。
- 聊天气泡、预览、代码、悄悄话、消息操作条和工具栏。
- 音频面板和玻璃效果语义变量。
- 滚动条、焦点、窄屏和减少动画设置。

`doc/examples/sealchat-ui-package-cyberpunk-neon.json` 是同一份 CSS 的可导入版本。它使用：

~~~json
"targets": ["global", "observer", "embed", "log-public"]
~~~

所以会覆盖全部非 Internal Surface。JavaScript 为空。

这个示例用于展示接口，不代表每个主题都要覆盖这么多内容。实际分发包应只保留自己真正需要维护的选择器。
