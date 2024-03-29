package fservers

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"sync"
	"syscall"
	"time"

	flog "github.com/farus422/fLogSystem"
	"github.com/fatih/color"
)

type IServerCallback interface {
	OnInit() bool
	OnRun() bool
	OnStop()
	OnShutdown()
}

type SServerFrame struct {
	exitChan      chan os.Signal
	serverWG      sync.WaitGroup
	logManager    *flog.SManager
	eventCallback IServerCallback
	ctx           context.Context
	cancel        context.CancelFunc

	// httpServer *http.Server
	// router     *mux.Router
}

func (sv *SServerFrame) Init(svcb IServerCallback) bool {
	sv.exitChan = make(chan os.Signal, 1)
	signal.Notify(sv.exitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL)
	sv.ctx, sv.cancel = context.WithCancel(context.Background())
	sv.logManager = flog.NewManager(sv.ctx, &sv.serverWG)
	sv.eventCallback = svcb
	if svcb != nil {
		return svcb.OnInit()
	}
	return true
}
func (sv *SServerFrame) Run() bool {
	if sv.eventCallback != nil {
		return sv.eventCallback.OnRun()
	}
	return true
}
func (sv *SServerFrame) Stop() {
	if sv.eventCallback != nil {
		sv.eventCallback.OnStop()
	}
}
func (sv *SServerFrame) Cancel() {
	sv.cancel()
}
func (sv *SServerFrame) Shutdown() {
	signal := &sShutdownSignal{msg: "shutdown"}
	sv.exitChan <- signal
}
func (sv *SServerFrame) GetLogManager() *flog.SManager {
	return sv.logManager
}
func (sv *SServerFrame) GetContext() context.Context {
	return sv.ctx
}
func (sv *SServerFrame) GetWaitGroup() *sync.WaitGroup {
	return &sv.serverWG
}

func (sv *SServerFrame) WaitForShutdown() {
	for {
		select {
		// 等待退出訊號
		case s, ok := <-sv.exitChan:
			if ok {
				Cprintf(color.FgWhite, "收到退出訊號：%s\n", s)
				// 告知業務處理函式該退出了
				if sv.eventCallback != nil {
					sv.eventCallback.OnShutdown()
				}
				if _, canceled := sv.logManager.Shutdown(4000, true); canceled == true {
					Cprintf(color.FgMagenta, "sv.logManager Shutdown timeout %d\n", 4000)
				}
				// 等待業務處理函式全都退出
				// sv.serverWG.Wait()
				sv.cancel()
				Cprintf(color.FgWhite, "伺服器關機程序全部完成，3秒後結束程序 . . .\n")
				time.Sleep(time.Second * 1)
				Cprintf(color.FgWhite, "2秒後結束程序 . . .\n")
				time.Sleep(time.Second * 1)
				Cprintf(color.FgWhite, "1秒後結束程序 . . .\n")
				time.Sleep(time.Second * 1)
				Cprintf(color.FgWhite, "程序結束\n")
			}
			return
		}
	}
}

func Cprintf(clr color.Attribute, format string, param ...interface{}) {
	if color.NoColor {
		fmt.Printf(fmt.Sprintf("\x1b[%dm%s\x1b[0m", clr, format), param...)
	} else {
		color.New(clr).Printf(format, param...)
	}
}

type sShutdownSignal struct {
	msg string
}

func (s *sShutdownSignal) String() string {
	return s.msg
}
func (s *sShutdownSignal) Signal() {
}
