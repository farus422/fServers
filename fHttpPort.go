package fservers

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	flog "github.com/farus422/fLogSystem"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type SHttpPort struct {
	serverWG  *sync.WaitGroup
	handlerWG sync.WaitGroup
	// httpServer *http.Server
	listener  net.Listener
	router    *mux.Router
	publisher *flog.SPublisher
}

func CORSHandler(handler http.Handler) http.Handler {
	originsOk := handlers.AllowedOrigins([]string{"*"})
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"})
	return handlers.CORS(originsOk, headersOk)(handler)
}

func (hp *SHttpPort) Init(wg *sync.WaitGroup, publisher *flog.SPublisher) {
	hp.serverWG = wg
	hp.router = mux.NewRouter()
	hp.publisher = publisher
}

func (hp *SHttpPort) Listen(portNo int) bool {
	hp.Unlisten()
	var err error
	hp.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", portNo))
	if err != nil {
		if hp.publisher != nil {
			hp.publisher.Publish(flog.Error("Failed to listen to port %d. err=%v", portNo, err))
		}
		return false
	}
	return true
}

func (hp *SHttpPort) Unlisten() {
	if hp.listener != nil {
		ln := hp.listener
		hp.listener = nil
		ln.Close()
	}
}

func (hp *SHttpPort) Serve() {
	go func() {
		hp.serverWG.Add(1)
		hp.handlerWG.Add(1)
		defer func() {
			hp.handlerWG.Done()
			hp.serverWG.Done()
		}()
		http.Serve(hp.listener, CORSHandler(hp.router))
	}()
}

func (hp *SHttpPort) ServeTLS(certFile, keyFile string) {
	go func() {
		hp.serverWG.Add(1)
		hp.handlerWG.Add(1)
		defer func() {
			hp.handlerWG.Done()
			hp.serverWG.Done()
		}()
		http.ServeTLS(hp.listener, CORSHandler(hp.router), certFile, keyFile)
	}()
}

// func (hp *SHttpPort) ListenAndServe(portNo int) bool {
// 	hp.httpServer = &http.Server{
// 		Addr:    fmt.Sprintf(":%d", portNo),
// 		Handler: CORSHandler(hp.router),
// 	}
// 	go func() {
// 		hp.serverWG.Add(1)
// 		hp.handlerWG.Add(1)
// 		defer func() {
// 			hp.handlerWG.Done()
// 			hp.serverWG.Done()
// 		}()
// 		if err := hp.httpServer.ListenAndServe(); err != nil {
// 			if err != http.ErrServerClosed {
// 				hp.publisher.Publish(flog.Error("httpServer.ListenAndServe failed! err=%v", err))
// 			}
// 			// } else {
// 			// 	hp.publisher.Publish(flog.NewLog(flog.LOGLEVELDebug, "SHttpPort closed")) ///////////////////////
// 			// }
// 			return
// 		}
// 	}()
// }
func (hp *SHttpPort) RouteFunc(path string, f func(http.ResponseWriter, *http.Request), methods ...string) {
	if len(methods) == 0 {
		hp.router.HandleFunc(path, hp.wrapHttpHandleFunc(f))
	} else {
		hp.router.HandleFunc(path, hp.wrapHttpHandleFunc(f)).Methods(methods...)
	}
}
func (hp *SHttpPort) WaitForAllDone() {
	hp.handlerWG.Wait()
}

func (hp *SHttpPort) Stop() {
	// hp.httpServer.Close()
	hp.Unlisten()
}

func (hp *SHttpPort) Shutdown() {
	// port.Unlisten()
	hp.Stop()
	hp.WaitForAllDone()
}
func (hp *SHttpPort) wrapHttpHandleFunc(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	// lgr := s.GetLogger()

	return func(w http.ResponseWriter, r *http.Request) {
		hp.handlerWG.Add(1)
		defer func() {
			if err := recover(); err != nil {
				if hp.publisher != nil {
					// log := flog.NewLog(flog.LOGLEVELError, "").AddPanicCallstack(0, "fServers.(*SHttpPort).wrapHttpHandleFunc")
					log := flog.Panic(flog.LOGLEVELError, "fServers.(*SHttpPort).wrapHttpHandleFunc", "")
					hp.publisher.Publish(log.SetCaption("%s() 發生panic, %v", log.GetFunctionName(), err))
				}
			}
			hp.handlerWG.Done()
		}()
		// dump, err := httputil.DumpRequest(r, true)
		// if err == nil {
		// 	lgr.Tracef("HttpRequest:\n%s\n", string(dump))
		// }
		// responseLogger := NewHttpResponseLogger(w, lgr)
		// f(responseLogger, r)
		f(w, r)
	}
}
