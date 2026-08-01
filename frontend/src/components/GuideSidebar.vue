<script setup>
import { ref } from 'vue'

const props = defineProps({
  // true = 常驻展开（出入库页）；false = 默认收起为窄条（物资页原行为）
  defaultOpen: { type: Boolean, default: true },
})

const open = ref(props.defaultOpen)
</script>

<template>
  <aside class="guide-sidebar" :class="{ open }">
    <button
      class="guide-toggle"
      @click="open = !open"
      :title="open ? '收起分类说明' : '分类说明'"
    >
      <span class="toggle-icon">{{ open ? '×' : '?' }}</span>
      <span v-if="!open" class="toggle-label">分类</span>
    </button>

    <div v-if="open" class="guide-body">
      <h3 class="guide-title">物资分类说明</h3>

      <div class="guide-section">
        <h4>电控 / Electronics</h4>
        <p>电路板、传感器、电源、控制器、通信设备。</p>
        <ul>
          <li>断路器：10A、20A、30A、40A</li>
          <li>配电板：PDP 1.0、PDP 2.0</li>
          <li>电源模块：迷你电源模块、VRM、降压模块、Radio Power Module</li>
          <li>传感器：CANcoder、CANrange、Pigeon 2.0、ThroughBore</li>
          <li>摄像头：Ardu Cam、OV9281、连接线</li>
          <li>控制器：Orange Pi 5pro/5plus、EMMC 模块</li>
          <li>通信：FRC Radio、CANivore、CANdi</li>
          <li>信号灯：Robot Signal Light</li>
          <li>拓展板：舵机拓展板、气动拓展板</li>
          <li>存储卡：32G SD</li>
          <li>电池负载测试仪 Battery Beak</li>
          <li>被动 POE 供电线缆 / 一分二线束</li>
        </ul>
        <p class="guide-note">⚠ 带"气动"字样的拓展板归电控，不是气动类。</p>
      </div>

      <div class="guide-section">
        <h4>电机 / Drivetrain</h4>
        <p>电机、减速箱和传动系统核心部件。</p>
        <ul>
          <li>减速箱：MAXPlanetary 3:1/4:1/5:1/X44/X60</li>
          <li>起步齿轮：8T/12T/14T/16T 等</li>
          <li>Kraken X60/X44 电机</li>
          <li>MAXPlanetary 输入套件、输入耦合器</li>
          <li>Kraken 螺丝端子盖套件</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>气动 / Pneumatics</h4>
        <ul>
          <li>气动拓展板（REV-11-1852）</li>
          <li>充气泵 Air Compressor 90C</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>轮子 / Wheels</h4>
        <p>所有轮子，不分材质和尺寸。</p>
        <ul>
          <li>Sushi Wheel（2in/3in/4in）</li>
          <li>Compliant Wheel（2in/3in/4in，35A-60A）</li>
          <li>Straight Flex Wheel（1.625in-4in）</li>
          <li>mk4i 轮组</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>传动件 / Power Transmission</h4>
        <p>齿轮、同步轮、链轮、轴承、轴套、耦合器。</p>
        <ul>
          <li>直齿轮：14T/26T/36T/66T 等（20DP）</li>
          <li>同步轮：24T/30T/36T/42T 等（HTD 5mm）</li>
          <li>链轮：#25/#35 各种齿数</li>
          <li>轴承：法兰轴承、ThunderHex、FR8ZZ 等</li>
          <li>轴套/衬套/抱轴器/垫圈轴套</li>
          <li>耦合器/适配器：六角耦合器、六角到圆形适配器</li>
          <li>六角梅花轴、凸轮 WCP</li>
        </ul>
        <p class="guide-note">⚠ 减速箱归"电机"类，齿轮归"传动件"类。</p>
      </div>

      <div class="guide-section">
        <h4>皮带/链条 / Belts &amp; Chains</h4>
        <ul>
          <li>HTD 同步带：5M-150 到 5M-1050（10mm/15mm 宽）</li>
          <li>链条：04c 链条</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>结构件 / Structure</h4>
        <p>支架、垫柱、垫片等结构支撑零件。</p>
        <ul>
          <li>ABS 隔离柱/垫柱/直通柱</li>
          <li>尼龙垫柱、铝合金垫柱</li>
          <li>垫圈/平垫（304 不锈钢）</li>
          <li>铆钉、连接环 M5</li>
          <li>铝合金外壳（Orange Pi 外壳）</li>
          <li>电梯滑块、注模相关</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>紧固件 / Fasteners</h4>
        <ul><li>四方螺母</li></ul>
        <p class="guide-note">库存极少，日常建议与"结构件"合并管理。</p>
      </div>

      <div class="guide-section">
        <h4>工具 / Tools</h4>
        <ul>
          <li>标签打印机、标签纸</li>
          <li>金属游标卡尺、手持称</li>
          <li>铆钉工具</li>
        </ul>
      </div>

      <div class="guide-section">
        <h4>耗材 / Consumables</h4>
        <ul>
          <li>胶水</li>
          <li>胶带：布基、绒布、玻璃纤维、透明</li>
          <li>端子：WAGO、SV、T 形三通、冷压、安德森</li>
          <li>热缩管</li>
          <li>电池：5 号/7 号</li>
        </ul>
      </div>

      <div class="guide-section guide-rules">
        <h4>快速判断规则</h4>
        <ol>
          <li>含"电机/电源/电路/控制/通信/传感器/摄像"→ <b>电控</b></li>
          <li>含"齿轮/链轮/同步轮/轴承/轴套/耦合"→ <b>传动件</b></li>
          <li>含"轮 wheel"→ <b>轮子</b></li>
          <li>含"皮带 5M HTD 链条"→ <b>皮带/链条</b></li>
          <li>不确定 → 选"其他"，以后再调整</li>
        </ol>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.guide-sidebar {
  position: relative;
  flex-shrink: 0;
  transition: width 0.2s ease;
  width: 36px;
  border-right: 1px solid var(--border);
}

.guide-sidebar.open {
  width: 320px;
}

.guide-toggle {
  position: absolute;
  top: 8px;
  right: -14px;
  width: 28px;
  height: 60px;
  border-radius: 0 8px 8px 0;
  background: var(--accent);
  border: none;
  color: #fff;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 4px;
  font-weight: 700;
  z-index: 2;
}

.guide-toggle:hover { opacity: 0.9; }

.guide-sidebar.open .guide-toggle {
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--muted);
  border-left: none;
}

.toggle-icon { font-size: 1.1rem; line-height: 1; }

.toggle-label {
  font-size: 0.65rem;
  writing-mode: vertical-rl;
  letter-spacing: 0.1em;
}

.guide-body {
  padding: 14px 10px 14px 14px;
  max-height: calc(100vh - 150px);
  overflow-y: auto;
  margin-right: 36px;
}

.guide-title {
  margin: 0 0 14px;
  font-size: 0.95rem;
  color: var(--accent);
}

.guide-section {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.guide-section:last-child { border-bottom: none; }

.guide-section h4 {
  margin: 0 0 4px;
  font-size: 0.85rem;
  color: var(--text);
}

.guide-section p {
  margin: 2px 0 6px;
  font-size: 0.78rem;
  color: var(--muted);
}

.guide-section ul,
.guide-section ol {
  margin: 4px 0;
  padding-left: 18px;
  font-size: 0.78rem;
  color: var(--muted);
}

.guide-section li { margin-bottom: 2px; }

.guide-note {
  color: var(--warn) !important;
  font-size: 0.75rem !important;
  margin-top: 6px !important;
}

.guide-rules {
  background: rgba(99,102,241,0.06);
  border-radius: 8px;
  padding: 10px 12px;
  border-bottom: none !important;
}

.guide-rules ol { padding-left: 20px; }
.guide-rules li { margin-bottom: 5px; font-size: 0.8rem; }
.guide-rules b { color: var(--accent); }
</style>
