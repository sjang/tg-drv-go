# TG-Drive-GO

A desktop file storage application that uses Telegram as a cloud storage backend. Upload, download, stream, and manage files through a modern desktop interface — all stored in your Telegram channels.

## Features

- **Telegram as Storage** — Each folder is backed by a private Telegram channel. When you create a folder, the app automatically creates a corresponding Telegram channel. Files uploaded to a folder are stored as messages in that channel, and folder sync pulls the latest state from Telegram.
- **Hidden Folders** — Folders can be toggled as hidden to keep them out of the default view. Hidden folders and their files remain accessible but are not displayed unless explicitly shown.
- **File Upload / Download** — Up to 2GB per file with real-time progress tracking
- **Video Streaming** — Stream video files directly in the browser via range requests
- **Smart Deduplication** — SHA256-based dedup avoids re-uploading identical files. If the same file already exists in another folder, it is forwarded instead of re-uploaded.
- **Folder Management** — Create, rename, delete, sync, and hide folders
- **Thumbnail Preview** — Auto-generated thumbnails for media files
- **2FA Support** — Full Telegram authentication including two-factor password
- **Cross-Platform** — Supports macOS and Windows

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop Framework | [Wails v2](https://wails.io/) |
| Backend | Go |
| Frontend | Svelte 3 + TypeScript + Vite |
| Telegram Client | [gotd/td](https://github.com/gotd/td) (pure Go, MTProto) |
| HTTP Router | [chi/v5](https://github.com/go-chi/chi) |
| Database | SQLite (modernc.org/sqlite) with WAL mode |
| Logging | [zap](https://github.com/uber-go/zap) |

## Project Structure

```
tg-drv-go/
├── main.go                     # Wails app entry point
├── app.go                      # App lifecycle & IPC bindings
├── internal/
│   ├── api/                    # HTTP API (chi router)
│   ├── config/                 # Configuration management
│   ├── hash/                   # SHA256 file hashing
│   ├── storage/                # SQLite database layer
│   ├── telegram/               # Telegram client & operations
│   └── thumb/                  # Thumbnail generation
├── frontend/
│   └── src/
│       ├── components/         # Svelte UI components
│       └── lib/                # API, stores, types
└── build/                      # Platform-specific build configs
```

## Prerequisites

- [Go](https://go.dev/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Installation

### 1. Clone the repository

```bash
git clone git@github.com:sjang/tg-drv-go.git
cd tg-drv-go
```

### 2. Get Telegram API credentials

You need a Telegram API ID and API Hash to use this application.

1. Go to [https://my.telegram.org](https://my.telegram.org)
2. Log in with your phone number
3. Click **"API development tools"**
4. Fill in the form (App title, Short name, etc.) and submit
5. You will receive your **api_id** and **api_hash**

### 3. Create configuration file

Create the config directory and file:

```bash
mkdir -p ~/.tg-drv
```

Create `~/.tg-drv/config.json` with the following content:

```json
{
  "telegram_api_id": YOUR_API_ID,
  "telegram_api_hash": "YOUR_API_HASH",
  "db_path": "~/.tg-drv/tgdrv.db",
  "http_port": 9876,
  "data_dir": "~/.tg-drv"
}
```

Replace `YOUR_API_ID` and `YOUR_API_HASH` with the values obtained from step 2.

### 4. Build and run

**Development mode:**

```bash
wails dev
```

**Production build:**

```bash
wails build
```

The built binary will be located in `build/bin/`.

## License

MIT

---

# TG-Drive-GO (한국어)

Telegram을 클라우드 저장소 백엔드로 사용하는 데스크톱 파일 관리 애플리케이션입니다. 모던 데스크톱 인터페이스를 통해 파일을 업로드, 다운로드, 스트리밍, 관리할 수 있으며, 모든 파일은 Telegram 채널에 저장됩니다.

## 주요 기능

- **Telegram을 저장소로 활용** — 각 폴더는 비공개 Telegram 채널에 매핑됩니다. 폴더를 생성하면 해당 Telegram 채널이 자동으로 생성되며, 업로드된 파일은 채널의 메시지로 저장됩니다. 폴더 동기화를 통해 Telegram의 최신 상태를 가져올 수 있습니다.
- **숨김 폴더** — 폴더를 숨김 상태로 전환하여 기본 화면에서 숨길 수 있습니다. 숨김 폴더와 파일은 그대로 유지되며, 명시적으로 표시하지 않는 한 목록에 나타나지 않습니다.
- **파일 업로드 / 다운로드** — 파일당 최대 2GB, 실시간 진행률 표시
- **비디오 스트리밍** — Range request를 통해 브라우저에서 직접 비디오 재생
- **스마트 중복 제거** — SHA256 기반으로 동일 파일의 중복 업로드를 방지합니다. 다른 폴더에 동일한 파일이 이미 존재하면 재업로드 없이 포워딩합니다.
- **폴더 관리** — 폴더 생성, 이름 변경, 삭제, 동기화, 숨김 처리
- **썸네일 미리보기** — 미디어 파일의 썸네일 자동 생성
- **2단계 인증 지원** — Telegram 2FA 비밀번호 인증 포함
- **크로스 플랫폼** — macOS 및 Windows 지원

## 기술 스택

| 레이어 | 기술 |
|--------|------|
| 데스크톱 프레임워크 | [Wails v2](https://wails.io/) |
| 백엔드 | Go |
| 프론트엔드 | Svelte 3 + TypeScript + Vite |
| Telegram 클라이언트 | [gotd/td](https://github.com/gotd/td) (순수 Go, MTProto) |
| HTTP 라우터 | [chi/v5](https://github.com/go-chi/chi) |
| 데이터베이스 | SQLite (modernc.org/sqlite), WAL 모드 |
| 로깅 | [zap](https://github.com/uber-go/zap) |

## 프로젝트 구조

```
tg-drv-go/
├── main.go                     # Wails 앱 진입점
├── app.go                      # 앱 라이프사이클 및 IPC 바인딩
├── internal/
│   ├── api/                    # HTTP API (chi 라우터)
│   ├── config/                 # 설정 관리
│   ├── hash/                   # SHA256 파일 해싱
│   ├── storage/                # SQLite 데이터베이스 레이어
│   ├── telegram/               # Telegram 클라이언트 및 작업
│   └── thumb/                  # 썸네일 생성
├── frontend/
│   └── src/
│       ├── components/         # Svelte UI 컴포넌트
│       └── lib/                # API, 스토어, 타입
└── build/                      # 플랫폼별 빌드 설정
```

## 사전 요구사항

- [Go](https://go.dev/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 설치 방법

### 1. 저장소 클론

```bash
git clone git@github.com:sjang/tg-drv-go.git
cd tg-drv-go
```

### 2. Telegram API 인증 정보 발급

이 애플리케이션을 사용하려면 Telegram API ID와 API Hash가 필요합니다.

1. [https://my.telegram.org](https://my.telegram.org) 에 접속합니다
2. 본인의 전화번호로 로그인합니다
3. **"API development tools"** 를 클릭합니다
4. 양식을 작성(App title, Short name 등)하고 제출합니다
5. **api_id**와 **api_hash**를 발급받습니다

### 3. 설정 파일 생성

설정 디렉토리와 파일을 생성합니다:

```bash
mkdir -p ~/.tg-drv
```

아래 내용으로 `~/.tg-drv/config.json` 파일을 생성합니다:

```json
{
  "telegram_api_id": YOUR_API_ID,
  "telegram_api_hash": "YOUR_API_HASH",
  "db_path": "~/.tg-drv/tgdrv.db",
  "http_port": 9876,
  "data_dir": "~/.tg-drv"
}
```

`YOUR_API_ID`와 `YOUR_API_HASH`를 2단계에서 발급받은 값으로 교체하세요.

### 4. 빌드 및 실행

**개발 모드:**

```bash
wails dev
```

**프로덕션 빌드:**

```bash
wails build
```

빌드된 바이너리는 `build/bin/` 디렉토리에 생성됩니다.

## 라이선스

MIT
