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
	OnInit()
	OnRun()
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

func (sv *SServerFrame) Init(svcb IServerCallback) {
	sv.exitChan = make(chan os.Signal, 1)
	signal.Notify(sv.exitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, os.Interrupt, os.Kill)
	sv.ctx, sv.cancel = context.WithCancel(context.Background())
	sv.logManager = flog.NewManager(sv.ctx, &sv.serverWG)
	sv.eventCallback = svcb
	if svcb != nil {
		svcb.OnInit()
	}
}
func (sv *SServerFrame) Run() {
	if sv.eventCallback != nil {
		sv.eventCallback.OnRun()
	}
}
func (sv *SServerFrame) Stop() {
	if sv.eventCallback != nil {
		sv.eventCallback.OnStop()
	}
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
		case <-sv.exitChan:
			PrintColorMsg(color.FgWhite, "get exit signal\n")
			// 告知業務處理函式該退出了
			if sv.eventCallback != nil {
				sv.eventCallback.OnShutdown()
			}
			if sv.logManager.Shutdown(4000) == false {
				PrintColorMsg(color.FgMagenta, "sv.logManager Shutdown timeout %d\n", 4000)
			}
			// 等待業務處理函式全都退出
			// sv.serverWG.Wait()
			sv.cancel()
			PrintColorMsg(color.FgWhite, "伺服器關機程序全部完成，3秒後返回")
			time.Sleep(time.Second * 3)
			return
		}
	}
}

func PrintColorMsg(clr color.Attribute, format string, param ...interface{}) {
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
