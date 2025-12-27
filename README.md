# GoPrintBridge

<p align="center">
  <img src="build/appicon.png" width="128" height="128" alt="GoPrintBridge">
</p>

<p align="center">
  <strong>Lightweight cross-platform printing bridge for silent browser printing.</strong><br>
  <em>Built with Go & Wails.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js" alt="Vue 3">
  <img src="https://img.shields.io/badge/Wails-v3%20Alpha-red" alt="Wails">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platform">
</p>

---

## 📖 Description

**GoPrintBridge** is a desktop application that acts as an HTTP bridge for silent printing. It allows browsers or kiosk applications to send print jobs via REST API to local printers without print dialogs.

### Key Features

- 🖨️ **Silent Print** - Print PDF and text without dialogs
- 🌐 **HTTP API** - Receive print jobs via REST endpoint
- 🎨 **Glassmorphism UI** - Modern interface with Vue 3 + Tailwind CSS
- �️ **Native System Tray** - Full system tray integration (Wails v3)
- �📋 **Printer Discovery** - Auto-detect available printers
- 💾 **Persistent Config** - Save settings in `config.yaml`
- 📝 **Logging** - Activity logs to `storage/logs/print.log`
- 🔔 **Toast Notifications** - Real-time UI notifications
- 🚀 **Auto Start** - Option to run on Windows login
- 📦 **Background Mode** - Minimize to system tray, server keeps running

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|------------|
| **Framework** | [Wails v3 (Alpha)](https://wails.io/) |
| **Backend** | Go 1.23 |
| **HTTP Server** | [Fiber](https://gofiber.io/) |
| **Config** | [Viper](https://github.com/spf13/viper) |
| **Logging** | [Zerolog](https://github.com/rs/zerolog) |
| **Frontend** | Vue 3 + Vite |
| **Styling** | Tailwind CSS |
| **Print (Windows)** | PowerShell + [alexbrainman/printer](https://github.com/alexbrainman/printer) |
| **Print (macOS/Linux)** | CUPS `lp` command |

---

## 📁 Project Structure

```
goprint-bridge/
├── main.go                 # Wails v3 entry point
├── app.go                  # Backend logic & Vue bindings
├── Taskfile.yml            # Build & Dev tasks
├── config.yaml             # App configuration
│
├── build/                  # Build config & assets
│   ├── config.yml          # Wails build config
│   └── Taskfile.yml        # Common tasks
│
├── config/                 # Config module (Viper)
│   └── config.go
│
├── logger/                 # Logging module (Zerolog)
│   └── logger.go
│
├── server/                 # HTTP server (Fiber)
│   └── server.go
│
├── printer/                # Silent print module
│   ├── printer_windows.go  # PowerShell + Spooler API
│   └── printer_unix.go     # CUPS lp command
│
├── autostart/              # Auto-start on login
│   ├── autostart.go        # macOS/Linux
│   └── autostart_windows.go # Windows Registry
│
├── storage/
│   └── logs/
│       └── print.log       # Log file
│
├── frontend/               # Vue 3 + Tailwind
│   ├── src/
│   │   ├── App.vue         # Main component
│   │   ├── style.css       # Tailwind + glass utilities
│   │   └── main.js
│   ├── tailwind.config.cjs
│   └── postcss.config.cjs
│
└── build/
    └── bin/                # Build output
```

---

## 🚀 Quick Start

### Prerequisites

- **Go** >= 1.23
- **Node.js** >= 16
- **Task** (go-task)
- **Wails v3 CLI**

```bash
# Install Task
go install github.com/go-task/task/v3/cmd/task@latest

# Install Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

### Development

```bash
# Clone repository
git clone https://github.com/rowjak/goprint-bridge.git
cd goprint-bridge

# Install dependencies and dev
task dev
```

### Build

```bash
# Build for current OS
task build

# Output: build/bin/goprint-bridge
```

---

## 📡 API Reference

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2024-12-26T19:30:00+07:00"
}
```

### Print Job

```http
POST /print
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "text",
  "content": "Hello World!"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `text`, `raw`, or `pdf` |
| `content` | string | Plain text or Base64-encoded PDF |

**Response (Success):**
```json
{
  "success": true,
  "message": "Print job completed"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Print failed: no printer selected"
}
```

---

## 💡 Usage Examples

### JavaScript (Browser/Kiosk)

```javascript
// Print text
fetch('http://localhost:9999/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    type: 'text',
    content: 'Hello from Kiosk!'
  })
})

// Print PDF (Base64)
const pdfBase64 = '...'; // Base64 encoded PDF
fetch('http://localhost:9999/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    type: 'pdf',
    content: pdfBase64
  })
})
```

### cURL

```bash
# Test print
curl -X POST http://localhost:9999/print \
  -H "Content-Type: application/json" \
  -d '{"type":"text","content":"Test Print from cURL"}'

# Health check
curl http://localhost:9999/health
```

---

## ⚙️ Configuration

File `config.yaml`:

```yaml
selected_printer: "EPSON_L3110_Series"
port: 9999
auto_start: false
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `selected_printer` | string | `""` | Selected printer name |
| `port` | int | `9999` | HTTP server port |
| `auto_start` | bool | `false` | Auto-start server when app opens |

---

## 📋 Vue Bindings (Frontend API)

Go functions callable from Vue:

| Function | Description |
|----------|-------------|
| `GetPrinters()` | Get printer list |
| `GetConfig()` | Get configuration |
| `SaveConfig(printer, port, autoStart)` | Save configuration |
| `StartServer(port)` | Start HTTP server |
| `StopServer()` | Stop server |
| `IsServerRunning()` | Check server status |
| `PrintTestPage()` | Print test page |
| `MinimizeToTray()` | Minimize to system tray |
| `QuitApp()` | Exit application |

---

## 📜 License

MIT License © 2024 [rowjak](https://github.com/rowjak)

---

## 🤝 Contributing

Pull requests welcome! For major changes, please open an issue first.

---

## 📞 Author

**Rozaq Abdur Rokhim**  
Email: rozaqabdur.rr@gmail.com
