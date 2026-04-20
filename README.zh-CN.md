# RawManga Downloader (生肉漫画下载器)
[English](./README.md)
---------------------


RawManga Downloader 是一款轻量级、跨平台的桌面应用程序，专为下载和阅读生肉漫画设计（目前主要支持 `klz9.com`）。本项目采用 Go 和 React 构建，为您提供从漫画解析到离线阅读的无缝体验。

![应用展示](./Design.png)

## 功能特性

- **漫画解析**：通过 URL 轻松解析漫画元数据和章节列表。
- **下载管理**：可靠的后台下载机制，支持实时进度查看。
- **集成阅读器**：
  - **卷轴模式**：连续垂直阅读。
  - **分页模式**：传统的左右翻页阅读。
  - 自动记忆每个章节的阅读进度。
- **本地书架**：使用嵌入式 SQLite 数据库管理您的已下载漫画收藏。
- **多语言支持**：完整支持英文、简体中文和日文。
- **现代 UI**：基于 Tailwind CSS 和 Lucide 图标构建的简洁、响应式界面。

## 技术栈

- **后端**: [Go](https://go.dev/) + [Wails v2](https://wails.io/) (桌面应用框架)
- **数据库**: [SQLite](https://www.sqlite.org/) (通过 `go-sqlite3`)
- **前端**: [React](https://reactjs.org/) + [TypeScript](https://www.typescriptlang.org/)
- **状态管理**: [Zustand](https://zustand-demo.pmnd.rs/) & [TanStack Query](https://tanstack.com/query/latest)
- **样式**: [Tailwind CSS](https://tailwindcss.com/)
- **构建工具**: [Vite](https://vitejs.dev/)

## 快速入门

### 环境准备

- [Go](https://go.dev/doc/install) (1.21 或更高版本)
- [Node.js](https://nodejs.org/) & [pnpm](https://pnpm.io/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### 开发模式

1. 克隆仓库：
   ```bash
   git clone https://github.com/sakagamijun/rawmanga-download-go.git
   cd rawmanga-download-go
   ```

2. 启动开发环境：
   ```bash
   wails dev
   ```

### 生产构建

编译适用于您当前操作系统的独立可执行文件：

```bash
wails build
```
编译产物将位于 `build/bin` 目录下。

## 鸣谢

- [Wails](https://wails.io/)：感谢它为 Go 和 Web 技术之间搭建的优秀桥梁。
- [shadcn/ui](https://ui.shadcn.com/)：提供了出色的 UI 设计灵感和基础组件。
- 感谢所有支持本项目开发的开源库。

## 开源协议

本项目采用 MIT 协议开源 - 详情请参阅 [LICENSE](LICENSE) 文件。
