// Category metadata + the keyword rules used to suggest a category while an
// item is being entered.
//
// The rules follow 分类说明.md and are ordered from most specific to least,
// because plenty of names match several groups:
//   - "同步轮" contains 轮 but belongs to 传动件
//   - "气动拓展板" contains 气动 but the guide puts it in 电控
//   - "电池负载测试仪" contains 电池 but belongs to 电控
// So each rule is checked in order and the first hit wins.
//
// Only names that exist in the live category list are suggested; a user who
// deleted a category must not be nudged towards it.

export const CATEGORY_META = {
  电控: {
    en: 'Electronics',
    desc: '电路板、传感器、电源、控制器、通信',
    keywords: [
      '电路', '电路板', '配电板', '控制板', '主板', '板卡', '传感器', 'sensor',
      '电源', 'power', '控制器', '通信', 'radio', '摄像头', 'camera', 'pdp',
      'pcm', 'vrm', '信号灯', 'sd卡', 'sd 卡', 'tf卡', 'tf 卡', '存储卡',
      '编码器', 'coder', 'beak', '负载测试', 'poe', 'orange pi', 'canivore',
      'candi', 'pigeon', 'canrange', 'throughbore', 'ardu', 'ov9281', '拓展板',
    ],
  },
  电机: {
    en: 'Drivetrain',
    desc: '电机、减速箱、传动核心部件',
    keywords: [
      '电机', 'motor', 'kraken', 'neo', '减速箱', 'gearbox', 'maxplanetary',
      'x44', 'x60', '输入套件', '端子盖',
    ],
  },
  传动件: {
    en: 'Power Transmission',
    desc: '齿轮、同步轮、链轮、轴承、轴套、耦合器',
    keywords: [
      '齿轮', 'gear', '同步轮', '同步带轮', 'pulley', '链轮', 'sprocket',
      '轴承', 'bearing', '轴套', 'bushing', '耦合', 'coupler', '六角轴', 'hex',
      '凸轮', 'cam', '抱轴器', '垫圈轴套', 'fr8zz', 'r6z', 'thunderhex',
    ],
  },
  '皮带/链条': {
    en: 'Belts & Chains',
    desc: 'HTD 同步带、链条',
    keywords: ['皮带', 'belt', 'htd', '5m-', '链条', 'chain', '04c'],
  },
  轮子: {
    en: 'Wheels',
    desc: '各种轮子，不分材质和尺寸',
    keywords: ['轮', 'wheel', 'mk4i', 'sushi', 'compliant', 'flex'],
  },
  // 分类说明.md 里“轴承”标注为存量少时并入传动件，所以默认分类里没有它。
  // 如果哪个队单独建了“轴承”分类，这条规则会优先命中。
  轴承: {
    en: 'Bearings',
    desc: '各类轴承',
    keywords: ['轴承', 'bearing', 'fr8zz', 'fr1810zz', 'r6z', 'thunderhex'],
  },
  气动: {
    en: 'Pneumatics',
    desc: '气动元件、充气泵',
    keywords: ['气动', 'pneumatic', '气泵', 'compressor', '气缸', '调压'],
  },
  结构件: {
    en: 'Structure',
    desc: '支架、垫柱、垫片、垫圈、铆钉、外壳',
    keywords: [
      '支架', 'bracket', '垫柱', 'standoff', '垫片', '平垫', 'spacer', '垫圈',
      'washer', '铆钉', 'rivet', '外壳', '滑块', '连接环', 'gusset',
      // 分类说明.md 建议把紧固件并入结构件管理；这些词只在没有单独
      // 配置“紧固件”分类时作为兜底（紧固件规则排在前面，优先命中）。
      '螺母', 'nut', '螺钉', 'screw', '螺栓', 'bolt',
    ],
  },
  紧固件: {
    en: 'Fasteners',
    desc: '螺母、螺钉、螺栓',
    keywords: ['螺母', 'nut', '螺钉', 'screw', '螺栓', 'bolt', '四方螺母'],
  },
  工具: {
    en: 'Tools',
    desc: '卡尺、称、手动工具',
    keywords: [
      '卡尺', 'caliper', '称', 'scale', '工具', 'tool', '扳手', '螺丝刀',
      '钳', '铆钉枪', '热熔枪', '胶枪',
    ],
  },
  耗材: {
    en: 'Consumables',
    desc: '胶水、胶带、端子、热缩管、电池',
    keywords: ['胶水', 'glue', '胶带', 'tape', '端子', 'terminal', 'wago', '热缩', '电池', 'battery', '焊', '扎带'],
  },
}

// Ordered: the first rule whose keyword appears in the text wins.
//
// 工具 / 结构件 / 紧固件 come first because they describe what the part *is*
// unambiguously ("铝合金外壳 Orange Pi" is a housing, even though it says
// Orange Pi; "铆钉枪" is a tool, even though it says 铆钉). The electronics,
// motor and wheel buckets have broad keywords, so they are checked after.
export const CATEGORY_RULES = [
  '工具',
  '紧固件',
  '结构件',
  '电控',
  '电机',
  '轴承',
  '传动件',
  '皮带/链条',
  '轮子',
  '气动',
  '耗材',
]

function normalize(value) {
  return String(value || '').toLowerCase()
}

/**
 * Suggest a category name for the given free text.
 *
 * @param {string} text  item name / spec / note typed so far
 * @param {string[]} available  names that actually exist, empty = no filtering
 * @returns {string} a category name, or '' when nothing matches
 */
export function suggestCategory(text, available = []) {
  const haystack = normalize(text).trim()
  if (!haystack) return ''

  const allowed = available.length ? new Set(available) : null

  for (const name of CATEGORY_RULES) {
    if (allowed && !allowed.has(name)) continue
    const meta = CATEGORY_META[name]
    if (!meta) continue
    if (meta.keywords.some((k) => haystack.includes(normalize(k)))) return name
  }
  return ''
}

/** Short description for a live category object from the API. */
export function categoryDesc(category) {
  if (!category) return ''
  return CATEGORY_META[category.name_zh]?.desc || ''
}
