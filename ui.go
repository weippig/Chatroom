package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChatUI struct {
	cr        *ChatRoom // 加了這個 attribute
	app       *tview.Application
	peersList *tview.TextView

	msgW    io.Writer
	inputCh chan string
	doneCh  chan struct{}
}

// 回傳一個 ChatUI 結構
// 呼叫 Run() 來執行
func NewChatUI(cr *ChatRoom) *ChatUI {
	app := tview.NewApplication()

	// 建立一個可以顯示聊天室訊息的 Box
	msgBox := tview.NewTextView()
	msgBox.SetDynamicColors(true)
	msgBox.SetBorder(true)
	msgBox.SetTitle(fmt.Sprintf("Room: %s", cr.roomName))

	// text views 是 io.Writers，沒辦法自動 refresh
	// 這裡添加了一個 change handler 讓 app 接收到新的訊息後會重新繪製
	msgBox.SetChangedFunc(func() {
		app.Draw()
	})

	// 建立使用者輸入的地方
	inputCh := make(chan string, 32)
	input := tview.NewInputField().
		SetLabel(cr.nick + " > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack)

	// 這邊設定了用戶按下 enter 或特定輸入值後發生的事
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			// 單純按 CRTL 不反應
			return
		}
		line := input.GetText()
		if len(line) == 0 {
			// 忽略空白行
			return
		}

		// bail if requested
		if line == "/quit" {
			app.Stop()
			return
		}

		// 將用戶的訊息加入 channel 後清空輸入區
		inputCh <- line
		input.SetText("")
	})

	// 建立一個顯示聊天室其他節點的視窗，這個區域會被 ui.refreshPeers function 刷新
	peersList := tview.NewTextView()
	peersList.SetBorder(true)
	peersList.SetTitle("Peers")
	peersList.SetChangedFunc(func() { app.Draw() })

	// chatPanel 是個水平排列的 box，裏面左側是 msgBox，右側是 peersList
	// peers list 佔 20 columns, 其他都是屬於 msgBox
	chatPanel := tview.NewFlex().
		AddItem(msgBox, 0, 1, false).
		AddItem(peersList, 20, 1, false)

	// flex 是垂直排列的 box，chatPanel 在上面，使用者輸入訊息的地方在下面
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatPanel, 0, 1, false).
		AddItem(input, 1, 1, true)

	app.SetRoot(flex, true)

	return &ChatUI{
		cr:        cr,
		app:       app,
		peersList: peersList,
		msgW:      msgBox,
		inputCh:   inputCh,
		doneCh:    make(chan struct{}, 1),
	}
}

// 在背景執行 handleEvents 的 for loop
// the event loop for the text UI.
func (ui *ChatUI) Run() error {
	go ui.handleEvents()
	defer ui.end() // 結束 Run() 後執行 end()

	return ui.app.Run()
}

// end signals the event loop to exit gracefully
func (ui *ChatUI) end() {
	ui.doneCh <- struct{}{}
}

func (ui *ChatUI) handleEvents() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()

	for {
		select {
		case input := <-ui.inputCh:
			// 使用者輸入訊息, publish 到 chatroom 並且印在 TUI 的 message window
			err := ui.cr.Publish(input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "publish error: %s", err)
			}
			ui.displaySelfMessage(input)

		case m := <-ui.cr.Messages:
			// when we receive a message from the chat room, print it to the message window
			ui.displayChatMessage(m)

		case <-peerRefreshTicker.C:
			// 週期性地刷新節點列表
			ui.refreshPeers()

		case <-ui.cr.ctx.Done():
			return

		case <-ui.doneCh:
			return
		}
	}
}

// displayChatMessage 把別人的訊息輸出在 message window,
// sender 的 nickname 用綠色 highlight.
func (ui *ChatUI) displayChatMessage(cm *ChatMessage) {
	prompt := withColor("green", fmt.Sprintf("<%s>:", cm.SenderNick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, cm.Message)
}

// displaySelfMessage 把自己的訊息輸出在 message window
// 自己的 nickname 用綠色 highlight.
func (ui *ChatUI) displaySelfMessage(msg string) { // 自己訊息
	prompt := withColor("yellow", fmt.Sprintf("<%s>:", ui.cr.nick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, msg)
}

// withColor 把 string 加上顏色後輸出在螢幕
func withColor(color, msg string) string {
	return fmt.Sprintf("[%s]%s[-]", color, msg)
}

// refreshPeers 得到 chat room 內的節點列表
// 把節點 peer id 最後 8 個字元展示在 ui 的 Peers panel
func (ui *ChatUI) refreshPeers() {
	peers := ui.cr.ps.ListPeers("chat-room:" + ui.cr.roomName)

	// clear is thread-safe
	ui.peersList.Clear()

	for _, p := range peers {
		pretty := p.Pretty()
		fmt.Fprintln(ui.peersList, pretty[len(pretty)-8:])
	}

	ui.app.Draw()
}
