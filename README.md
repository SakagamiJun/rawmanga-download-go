# RawManga Downloader
[中文版](./README.zh-CN.md)
---------------------


RawManga Downloader is a lightweight, cross-platform desktop application designed for downloading and reading manga from raw manga providers (primarily supporting `klz9.com`). Built with Go and React, it provides a seamless experience from discovery to offline reading.

![Application Showcase](./Design.png)

## Features

- **Manga Parsing**: Effortlessly resolve manga metadata and chapter lists from URLs.
- **Download Manager**: Reliable background downloading with real-time progress tracking.
- **Integrated Reader**: 
  - **Scroll Mode**: Continuous vertical reading.
  - **Paged Mode**: Traditional side-to-side reading.
  - Opens local chapter folders and chapter `.zip` / `.cbz` archives directly.
  - Remembers your reading progress for every chapter.
- **Local Library**: Manage your downloaded manga collection with an embedded SQLite database.
- **Internationalization**: Full support for English, Simplified Chinese, and Japanese.
- **Modern UI**: A clean, responsive interface built with Tailwind CSS and Lucide icons.

## Tech Stack

- **Backend**: [Go](https://go.dev/) + [Wails v2](https://wails.io/) (Desktop Framework)
- **Database**: [SQLite](https://www.sqlite.org/) (via `go-sqlite3`)
- **Frontend**: [React](https://reactjs.org/) + [TypeScript](https://www.typescriptlang.org/)
- **State Management**: [Zustand](https://zustand-demo.pmnd.rs/) & [TanStack Query](https://tanstack.com/query/latest)
- **Styling**: [Tailwind CSS](https://tailwindcss.com/)
- **Build Tool**: [Vite](https://vitejs.dev/)

## Getting Started

### Prerequisites

- [Go](https://go.dev/doc/install) (1.21 or later)
- [Node.js](https://nodejs.org/) & [pnpm](https://pnpm.io/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Development

1. Clone the repository:
   ```bash
   git clone https://github.com/sakagamijun/rawmanga-download-go.git
   cd rawmanga-download-go
   ```

2. Run in development mode:
   ```bash
   wails dev
   ```

### Building for Production

To create a standalone executable for your operating system:

```bash
wails build
```
The binary will be located in the `build/bin` directory.

## Acknowledgements

- [Wails](https://wails.io/) for the amazing bridge between Go and Web technologies.
- [shadcn/ui](https://ui.shadcn.com/) for the inspiration and base components.
- All the open-source libraries that made this project possible.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
