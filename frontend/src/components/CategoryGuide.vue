<script setup>
// Content of the "分类说明" drawer. Kept as data instead of long markup so the
// drawer stays readable and the sections can be scanned quickly while working.
const sections = [
  {
    title: '电控 / Electronics',
    summary: '电路板、传感器、电源、控制器、通信设备。',
    items: [
      '断路器：10A / 20A / 30A / 40A',
      '配电板：PDP 1.0 / PDP 2.0',
      '电源模块：迷你电源模块、VRM、降压模块、Radio Power Module',
      '传感器：CANcoder、CANrange、Pigeon 2.0、ThroughBore',
      '摄像头：Ardu Cam、OV9281、连接线',
      '控制器：Orange Pi 5pro/5plus、EMMC 模块',
      '通信：FRC Radio、CANivore、CANdi',
      '信号灯：Robot Signal Light',
      '拓展板：舵机拓展板、气动拓展板',
      '存储卡：32G SD；电池负载测试仪 Battery Beak',
      '被动 POE 供电线缆 / 一分二线束',
    ],
    note: '带“气动”字样的拓展板归电控，不是气动类。',
  },
  {
    title: '电机 / Drivetrain',
    summary: '电机、减速箱和传动系统核心部件。',
    items: [
      '减速箱：MAXPlanetary 3:1 / 4:1 / 5:1、X44、X60',
      '起步齿轮：8T / 12T / 14T / 16T 等',
      'Kraken X60 / X44 电机',
      'MAXPlanetary 输入套件、输入耦合器',
      'Kraken 螺丝端子盖套件',
    ],
  },
  {
    title: '气动 / Pneumatics',
    items: ['气动拓展板（REV-11-1852）', '充气泵 Air Compressor 90C'],
  },
  {
    title: '轮子 / Wheels',
    summary: '所有轮子，不分材质和尺寸。',
    items: [
      'Sushi Wheel（2in / 3in / 4in）',
      'Compliant Wheel（2in–4in，35A–60A）',
      'Straight Flex Wheel（1.625in–4in）',
      'mk4i 轮组',
    ],
  },
  {
    title: '传动件 / Power Transmission',
    summary: '齿轮、同步轮、链轮、轴承、轴套、耦合器。',
    items: [
      '直齿轮：14T / 26T / 36T / 66T 等（20DP）',
      '同步轮：24T / 30T / 36T / 42T 等（HTD 5mm）',
      '链轮：#25 / #35 各种齿数',
      '轴承：法兰轴承、ThunderHex、FR8ZZ、R6Z 等',
      '轴套/衬套/抱轴器/垫圈轴套',
      '耦合器/适配器：六角耦合器、六角到圆形适配器',
      '六角梅花轴、凸轮 WCP',
    ],
    note: '减速箱归“电机”类，齿轮归“传动件”类。',
  },
  {
    title: '皮带/链条 / Belts & Chains',
    items: ['HTD 同步带：5M-150 到 5M-1050（10mm / 15mm 宽）', '链条：04c 链条'],
  },
  {
    title: '结构件 / Structure',
    summary: '支架、垫柱、垫片等结构支撑零件。',
    items: [
      'ABS 隔离柱 / 垫柱 / 直通柱',
      '尼龙垫柱、铝合金垫柱',
      '垫圈 / 平垫（304 不锈钢）',
      '铆钉、连接环 M5',
      '铝合金外壳（Orange Pi 外壳）',
      '电梯滑块、注模相关',
    ],
  },
  {
    title: '紧固件 / Fasteners',
    items: ['四方螺母'],
    note: '库存极少，日常建议与“结构件”合并管理。',
  },
  {
    title: '工具 / Tools',
    items: ['标签打印机、标签纸', '金属游标卡尺、手持称', '铆钉工具'],
  },
  {
    title: '耗材 / Consumables',
    items: [
      '胶水',
      '胶带：布基、绒布、玻璃纤维、透明',
      '端子：WAGO、SV、T 形三通、冷压、安德森',
      '热缩管',
      '电池：5 号 / 7 号',
    ],
  },
]

const rules = [
  '含“电机/电源/电路/控制/通信/传感器/摄像”→ 电控',
  '含“齿轮/链轮/同步轮/轴承/轴套/耦合”→ 传动件',
  '含“轮 wheel”→ 轮子',
  '含“皮带 5M HTD 链条”→ 皮带/链条',
  '不确定 → 选“其他”，以后再调整',
]
</script>

<template>
  <div class="guide">
    <div v-for="s in sections" :key="s.title" class="guide-section">
      <h4>{{ s.title }}</h4>
      <p v-if="s.summary" class="guide-summary">{{ s.summary }}</p>
      <ul>
        <li v-for="item in s.items" :key="item">{{ item }}</li>
      </ul>
      <p v-if="s.note" class="guide-note">⚠ {{ s.note }}</p>
    </div>

    <div class="guide-rules">
      <h4>快速判断</h4>
      <ol>
        <li v-for="r in rules" :key="r">{{ r }}</li>
      </ol>
    </div>
  </div>
</template>

<style scoped>
.guide-section {
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--border);
}

.guide-section:last-of-type {
  border-bottom: none;
}

.guide-section h4 {
  margin-bottom: 5px;
  font-size: 0.9375rem;
  color: var(--text);
}

.guide-summary {
  margin: 0 0 7px;
  font-size: 0.875rem;
  color: var(--muted);
}

.guide-section ul {
  margin: 0;
  padding-left: 18px;
  font-size: 0.875rem;
  color: var(--text-soft);
}

.guide-section li {
  margin-bottom: 4px;
}

.guide-note {
  margin: 8px 0 0;
  font-size: 0.8125rem;
  color: var(--warn);
}

.guide-rules {
  padding: 14px 16px;
  border-radius: var(--radius);
  background: var(--accent-soft);
}

.guide-rules h4 {
  margin-bottom: 8px;
  font-size: 0.875rem;
  color: var(--accent);
}

.guide-rules ol {
  margin: 0;
  padding-left: 18px;
  font-size: 0.875rem;
  color: var(--text-soft);
}

.guide-rules li {
  margin-bottom: 6px;
}
</style>
