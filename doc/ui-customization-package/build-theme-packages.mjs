// 把 theme-<slug>.css / theme-<slug>.js 打包成可导入的 sealchat.ui-package JSON。
// 用法：node doc/ui-customization-package/build-theme-packages.mjs
import { readFileSync, writeFileSync, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = dirname(fileURLToPath(import.meta.url))
const TARGETS = ['global', 'observer', 'embed', 'log-public']

const THEMES = [
  {
    slug: 'neon-runner',
    name: 'NEON//RUNNER 霓虹夜行',
    description: '赛博朋克主题：酸性黄 + 冰青 + 警报红，切角面板、扫描线、透视网格地面、终端风输入区。脚本提供顶栏 HUD（时钟 / 节点 / 接收计数）、每会话一次的开机序列、新消息信号到达动画与频道标题故障效果。夜间「霓虹夜行」，日间「镀铬白昼」。',
  },
  {
    slug: 'rlyeh-archive',
    name: "R'LYEH ARCHIVE 拉莱耶档案馆",
    description: '克苏鲁的呼唤主题：1920 年代调查员档案。场内为钢笔手记，场外为铅笔旁注，头像是贴胶带的旧照片，频道是图书馆索引卡。脚本提供深海雾气、角落触须、注视之眼、理智值（SAN）氛围仪表、禁忌词蠕动效果与 d100 大成功 / 大失败印章。夜间「深渊」，日间「档案」。',
  },
  {
    slug: 'triangle-agency',
    name: 'TRIANGLE AGENCY 三角机构',
    description: '《三角机构》办公室新怪谈主题：员工手册式白纸、黑色细线与机构红三角，等宽编号，剧透显示为涂黑。脚本提供顶栏工牌 HUD（时钟、按频道累计的混沌计数）、6d4 判定解析（出 3 的成功数、混沌、三重升华不增加混沌）与极低频率的内部备忘。日间「办公时间」，夜间「加班时间」。',
  },
]

for (const theme of THEMES) {
  const cssPath = join(dir, `theme-${theme.slug}.css`)
  const jsPath = join(dir, `theme-${theme.slug}.js`)
  const doc = {
    format: 'sealchat.ui-package',
    formatVersion: 1,
    runtimeApiVersion: 1,
    manifest: { name: theme.name, description: theme.description },
    targets: TARGETS,
    css: readFileSync(cssPath, 'utf8'),
    js: existsSync(jsPath) ? readFileSync(jsPath, 'utf8') : '',
  }
  const out = join(dir, `sealchat-ui-package-${theme.slug}.json`)
  writeFileSync(out, `${JSON.stringify(doc, null, 2)}\n`)
  const kib = n => (Buffer.byteLength(n, 'utf8') / 1024).toFixed(1)
  console.log(`${out}  css ${kib(doc.css)} KiB  js ${kib(doc.js)} KiB`)
}
