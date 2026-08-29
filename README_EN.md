<div align="center">
  <img src="assets/banner.jpg" alt="Q-Solver Banner" width="100%" style="border-radius: 16px; box-shadow: 0 10px 40px rgba(0,0,0,0.15);">

  <br>

  <h1>🧠 Q-Solver</h1>
  
  <h3>AI-Powered Real-Time Desktop Assistant · Screen Analysis · Voice Chat</h3>
  
  <p><i>🎯 Snapshot → Think → Solve. Your invisible AI Co-pilot.</i></p>

  <p>
    <a href="https://github.com/kd14125/Q-solver/stargazers"><img src="https://img.shields.io/github/stars/kd14125/Q-solver?color=ffcb6b&style=for-the-badge&labelColor=30363d" alt="Stars"></a>
    <a href="https://github.com/kd14125/Q-solver/releases"><img src="https://img.shields.io/github/v/release/kd14125/Q-solver?color=89d185&style=for-the-badge&labelColor=30363d" alt="Release"></a>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white&labelColor=30363d" alt="Go">
    <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=for-the-badge&logo=vue.js&logoColor=white&labelColor=30363d" alt="Vue">
    <img src="https://img.shields.io/badge/Wails-v2-E30613?style=for-the-badge&logo=wails&logoColor=white&labelColor=30363d" alt="Wails">
  </p>
  
  <p>
    <img src="https://img.shields.io/badge/Windows-0078D6?style=flat-square&logo=windows&logoColor=white" alt="Windows">
  </p>

  <br>

  <p>
    <a href="#-features">Features</a> •
    <a href="#-quick-start">Install</a> •
    <a href="#-demo">Demo</a> •
    <a href="#-shortcuts">Shortcuts</a> •
    <a href="README.md">中文文档</a>
  </p>
  
  <br>
  
  <img src="assets/demo.gif" alt="Demo" width="92%" style="border-radius: 12px; box-shadow: 0 12px 40px rgba(0,0,0,0.25); border: 1px solid rgba(255,255,255,0.1);">

</div>

<br>
<br>

> [!CAUTION]
> **🚧 Development Status**: This project is currently in **Pre-Alpha**. Features may change significantly. Proceed with caution.

<br>

<div align="center">

## 🌟 Core Highlights

</div>

<table>
<tr>
<td width="50%" valign="top">

### 🖼️ Instant Screen Solving
Capture any part of your screen and get an instant AI analysis with a single hotkey.
- **📸 Smart Recognition**: Accurately recognizes text, math formulas, and code.
- **🧠 Deep Thinking**: Powered by extensive reasoning models like o1 and Claude 3.5.
- **⚡️ Zero Distraction**: Floating ghost window designed not to interrupt your flow.

</td>
<td width="50%" valign="top">

### 🎙️ Immersive Voice Chat
Integrated with Google Gemini Live API for a seamless real-time conversation experience.
- **🗣️ Natural Interaction**: Millisecond latency, feels just like a human call.
- **🗺️ Auto Mind Map**: Visualizes your conversation structure automatically.
- **📝 Smart Notes**: Auto-transcribes and summarizes key points.

</td>
</tr>
</table>

<br>

<div align="center">

## ✨ Core Features

</div>

### 🛡️ Stealth Mode

Designed for privacy and multitasking, offering a "Ghost Window" experience.

> ⚠️ **Note**: Please test the actual effect yourself.

| Feature | Description |
|:---|:---|
| **🚫 Recording Proof** | Invisible to most screen recording/sharing software. |
| **👻 Click-Through** | Enable to interact with content behind the window seamlessly. |
| **📌 Always on Top** | Floats above all other windows for quick reference. |
| **🔕 Focus Guard** | Intelligently manages window focus to avoid stealing keystrokes. |

---

### 🧠 Model Ecosystem

**Supports OpenAI / Gemini / Claude / DeepSeek (Custom) and more.**

- **Live API**: Experience millisecond-latency voice chat with Gemini 2.0.
- **Custom Models**: Compatible with any OpenAI format API.

---

<br>

<div align="center">

## 📸 Interface Showcase

</div>

| | | |
|:---:|:---:|:---:|
| <img src="assets/img1.png" width="100%" style="border-radius: 8px;"/> | <img src="assets/img6.png" width="100%" style="border-radius: 8px;"/> | <img src="assets/img7.png" width="100%" style="border-radius: 8px;"/> |

<br>
<br>

## 🚀 Quick Start

### 📥 Option 1: Download App (Recommended)

Get the latest Windows build from the [Releases Page](https://github.com/kd14125/Q-solver/releases). This enhanced fork currently builds and maintains Windows only.

### 🛠️ Option 2: Build from Source

**Prerequisites**: Go 1.25+, Node.js 22+, Wails CLI

```bash
# 1. Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. Clone repo
git clone https://github.com/kd14125/Q-solver.git
cd Q-Solver

# 3. Dev mode (Hot Reload)
wails dev

# 4. Build Production
wails build -ldflags "-s -w" -tags prod
```

<br>

## ⌨️ Shortcuts

> 💡 **Tip**: Windows shortcuts are customizable. Defaults are shown below.

| Action | Windows Default |
|:---|:---:|
| **Capture screenshot (1–3)** 📸 | `F8` |
| **Send screenshots and request answer** 🤖 | `F7` |
| **Toggle Visibility** 👁️ | `F9` |
| **Toggle Click-Through** 👻 | `F10` |
| **Nudge Window** ↕️ | `Alt + Arrows` |
| **Fast Scroll** 📜 | `Alt + PgUp/Dn` |

<br>

## ⚙️ Configuration

1. Click the **Settings** icon (top-right).
2. Select text **Provider** (e.g., Gemini, OpenAI).
3. Paste your **API Key**.
4. (Optional) Enable **Live API** for voice features.

## 🛠️ Tech Stack

- **Core**: [Go](https://go.dev/) (Logic) + [Wails](https://wails.io/) (Binding)
- **UI**: [Vue 3](https://vuejs.org/) + [Vue Flow](https://vueflow.dev/) (Mind Map)
- **AI**: Gemini Protocol, OpenAI SDK
- **Audio**: Windows WASAPI Loopback (via malgo)

<br>

## 📈 Star History

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

## 📄 License

Distributed under the **CC BY-NC 4.0** License. Intended for **personal, non-commercial use only**.

---

<div align="center">
  <p>Made with ❤️ by <a href="https://github.com/jym66">jym66</a></p>
  <p>
    If you enjoy using Q-Solver, please leave a <b>⭐ Star</b>!
  </p>
</div>
