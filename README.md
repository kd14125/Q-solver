<div align="center">
  <img src="assets/banner.jpg" alt="Q-Solver Banner" width="100%" style="border-radius: 16px; box-shadow: 0 10px 40px rgba(0,0,0,0.15);">

  <br>

  <h1>🧠 Q-Solver</h1>
  
  <h3>AI 驱动的实时桌面助手 · 截图解题 · 实时语音面试辅助</h3>
  
  <p><i>🎯 一键截图开启深度思考，实时语音连接智能未来</i></p>

  <p>
    <a href="https://github.com/kd14125/Q-solver/stargazers"><img src="https://img.shields.io/github/stars/kd14125/Q-solver?color=ffcb6b&style=for-the-badge&labelColor=30363d" alt="Stars"></a>
    <a href="https://github.com/kd14125/Q-solver/releases"><img src="https://img.shields.io/github/v/release/kd14125/Q-solver?color=89d185&style=for-the-badge&labelColor=30363d" alt="Release"></a>
    <img src="https://img.shields.io/badge/version-v1.3.0-10b981?style=for-the-badge&labelColor=30363d" alt="Version 1.3.0">
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white&labelColor=30363d" alt="Go">
    <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=for-the-badge&logo=vue.js&logoColor=white&labelColor=30363d" alt="Vue">
    <img src="https://img.shields.io/badge/Wails-v2-E30613?style=for-the-badge&logo=wails&logoColor=white&labelColor=30363d" alt="Wails">
  </p>
  
  <p>
    <img src="https://img.shields.io/badge/Windows-0078D6?style=flat-square&logo=windows&logoColor=white" alt="Windows">
  </p>

  <br>

  <p>
    <a href="#-核心特性">特性</a> •
    <a href="#-快速开始">安装</a> •
    <a href="#-界面展示">演示</a> •
    <a href="#%EF%B8%8F-配置与使用">配置</a> •
    <a href="CHANGELOG.md">更新日志</a> •
    <a href="README_EN.md">English Documentation</a>
  </p>
  
  <br>
  
  <img src="assets/demo.gif" alt="Demo" width="92%" style="border-radius: 12px; box-shadow: 0 12px 40px rgba(0,0,0,0.25); border: 1px solid rgba(255,255,255,0.1);">

</div>

<br>
<br>

> [!CAUTION]
> **🚧 开发阶段警告**：本项目目前处于**早期开发预览阶段 (Pre-Alpha)**。功能可能会随版本更新发生重大变化，建议仅用于测试和尝鲜。

<br>

## 🆕 v1.3.0 更新内容

- **快捷键稳定性重构**：修复录制 F1、修改透明度或保存设置后 F7/F8/F9、Alt 方向键失效的问题，Windows Hook 现在拥有完整的启动、停止与重启生命周期。
- **遗漏抬键自动恢复**：即使系统或第三方软件吞掉 `KEYUP`，也会在下一次按键前清理残留状态，不再把后续快捷键误判成组合键。
- **透明界面升级**：顶部栏、历史栏、回答区、欢迎页和状态组件支持高透明/特别透明效果，AI 回答和代码块背景可完全透明。
- **纯白回答样式**：AI 正文、标题、列表、行内代码、代码块和行号统一使用白色，移除绿色代码胶囊和语法高亮背景。
- **可隐藏界面内容**：设置中可隐藏顶部工具栏和历史问题栏；新增 `F6` 总开关，同时隐藏欢迎页内容和加载提示，并可一键恢复。
- **回答显示调节**：新增 AI 文字透明度滑块，修正透明度方向，并兼容迁移早期错误保存的数值。
- **简洁截图反馈**：连续截图后右下角只显示 `1/3`、`2/3`、`3/3`，不再展示缩略图和操作框。
- **移除等待动画**：发送问题后不再显示“深度思考中”圆环、计时器或追加回复动画。
- **单实例与配置迁移**：避免同时启动多个 Q-Solver；配置固定保存到 `%APPDATA%/Q-Solver/config.json`，并自动迁移旧版配置。

完整记录请查看 [CHANGELOG.md](CHANGELOG.md)。

> [!NOTE]
> 当前增强版只构建、发布和维护 Windows 版本。

<br>

<div align="center">

## 🌟 核心亮点

</div>

<table>
<tr>
<td width="50%" valign="top">

### 🖼️ 极速截图求解
只需一个快捷键，即刻捕获屏幕内容并进行 AI 分析。
- **📸 智能识别**：精准识别文字、公式、代码。
- **🧠 深度思考**：支持 o1, Claude 3.5 等强推理模型。
- **⚡️ 零干扰**：悬浮窗设计，不打断当前工作流。

</td>
<td width="50%" valign="top">

### 🎙️ 实时语音面试辅助
支持 Qwen3.5 Omni Realtime 与原有 Gemini Live，在面试官提问结束后生成候选人可直接参考的文字回答。
- **🔊 系统声音内录**：Windows 监听默认播放设备，不直接监听麦克风。
- **🧠 Semantic VAD**：自动识别完整提问和追问，无需固定时长切段。
- **💬 口语化短回答**：默认控制在适合面试现场快速阅读和复述的长度。

</td>
</tr>
</table>

<br>

<div align="center">

## ✨ 核心特性

</div>

### 🛡️ 隐身模式 (Stealth Mode)

专为隐私与多任务设计，打造“幽灵”般的窗口体验。

> ⚠️ **提示**：具体效果请自行测试。

| 特性 | 描述 |
|:---|:---|
| **🚫 防录屏检测** | 窗口对大多数录屏/截屏软件不可见 |
| **👻 鼠标穿透** | 开启后可透过窗口点击后方内容，互不影响 |
| **📌 全局置顶** | 始终悬浮在其他窗口之上，重要信息一眼即达 |
| **🔕 沉浸免打扰** | 精心设计的焦点管理，输入时不抢占主窗口焦点 |

---

### 🧠 多模型生态

**截图答题支持 OpenAI-compatible / Gemini / Claude / DeepSeek（自定义）等主流模型。**

- **Qwen Realtime**：支持 `qwen3.5-omni-plus-realtime` 实时转录与文字回答。
- **Gemini Live**：保留原有 Gemini Live 能力。
- **自定义模型**: 兼容所有 OpenAI 格式的 API 接口。
- **配置隔离**：截图模型与语音模型分别保存 API Key、模型、提示词和生成参数。



<br>

<div align="center">

## 📸 界面展示

</div>

| | | |
|:---:|:---:|:---:|
| <img src="assets/img1.png" width="100%" style="border-radius: 8px;"/> | <img src="assets/img6.png" width="100%" style="border-radius: 8px;"/> | <img src="assets/img7.png" width="100%" style="border-radius: 8px;"/> |

<br>
<br>

## 🚀 快速开始

### 📥 方式一：直接下载 (如果你想直接使用)

前往 [Releases 页面](https://github.com/kd14125/Q-solver/releases) 下载最新的 Windows 版本。

### 🛠️ 方式二：源码构建 (如果你是开发者)

**环境要求**：Go 1.25+, Node.js 22+, Wails CLI

```bash
# 1. 安装 Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. 克隆仓库
git clone https://github.com/kd14125/Q-solver.git
cd Q-Solver

# 3. 开发模式运行 (支持热重载)
wails dev

# 4. 编译发布版本
wails build -ldflags "-s -w" -tags prod
```

<br>

## ⌨️ 快捷键指南

> 💡 **提示**：Windows 支持自定义快捷键，下表为默认按键。

| 动作 | Windows 默认快捷键 |
|:---|:---:|
| **截图（可连续 1–3 张）** 📸 | `F8` |
| **发送截图并请求回答** 🤖 | `F7` |
| **隐藏/显示界面内容** 🫥 | `F6` |
| **显示/隐藏窗口** 👁️ | `F9` |
| **切换鼠标穿透** 👻 | `F10` |
| **微调窗口位置** ↕️ | `Alt + 方向键` |
| **快速翻页** 📜 | `Alt + PgUp/Dn` |

<br>

## ⚙️ 配置与使用

1. 点击窗口右上角的 **设置** 图标。
2. 在 **截图答题模型** 中配置 Provider、API Key、Base URL、模型和提示词。
3. 如需语音面试辅助，在 **语音面试模型** 中单独配置 Qwen Realtime API Key、Workspace ID、模型和提示词。
4. Qwen Realtime 与旧版第三方 STT 模式不能同时启用。
5. 在 **常规设置** 中可切换日夜主题，调整窗口透明度、AI 文字透明度、AI 字体大小和代码块自动换行。
6. 可分别隐藏顶部工具栏和历史问题栏，也可用默认快捷键 `F6` 一键隐藏或恢复界面内容。

> [!IMPORTANT]
> API Key 仅应填写在本机设置中。不要把 API Key 写入 README、截图、Issue 或提交到 Git 仓库。

## 🛠️ 技术栈概览

- **Core**: [Go](https://go.dev/) (Logic) + [Wails](https://wails.io/) (Binding)
- **UI**: [Vue 3](https://vuejs.org/) + [Vue Flow](https://vueflow.dev/) (Mind Map)
- **AI**: Qwen Realtime, Gemini Protocol, OpenAI-compatible API
- **Audio**: Windows WASAPI Loopback（via malgo）

<br>

<br>

## 📈 Star 趋势

<div align="center">
  <a href="https://star-history.com/#kd14125/Q-solver&Date">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=kd14125/Q-solver&type=Date&theme=dark" />
      <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=kd14125/Q-solver&type=Date" />
      <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=kd14125/Q-solver&type=Date" />
    </picture>
  </a>
</div>

<br>

## 📄 许可证

本项目基于 **CC BY-NC 4.0** 协议开源，仅供 **非商业个人学习与研究** 使用。

---

<div align="center">
  <p>基于 <a href="https://github.com/jym66/Q-solver">jym66/Q-solver</a>，增强版由 <a href="https://github.com/kd14125">kd14125</a> 维护。</p>
  <p>
    如果你觉得这个项目有趣，欢迎点个 <b>⭐ Star</b> 支持一下！
  </p>
</div>
