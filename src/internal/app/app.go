package app

import (
    "context"
    "errors"
    "fmt"
    "golang.org/x/sync/errgroup"
    "os"
    "os/signal"
    // "regexp"
    "syscall"
    "net/http"
    // "time"

    // "package/main/internal/xx"
    // "package/main/internal/yy"
)

var ctx, cancel = context.WithCancel(context.Background())
var group, groupCtx = errgroup.WithContext(ctx)
var server *http.Server

// Run is a like main function
func Run() { 
    log.Info("Starting app")

    server = &http.Server{Addr: conf.HTTPListenIPPort, Handler: nil}

    group.Go(func() error {
        signalChannel := make(chan os.Signal, 1)
        defer close(signalChannel)
        signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
        select {
        case sig := <-signalChannel:
            log.Errorf("Received signal: %s", sig)
            if err := server.Shutdown(ctx); err != nil {
                log.Errorf("Received an error while shutting down the server: %s", err)
            }
            cancel()
        case <-groupCtx.Done():
            log.Error("Closing signal goroutine")
            if err := server.Shutdown(ctx); err != nil {
                log.Errorf("Received an error while shutting down the server: %s", err)
            }
            return groupCtx.Err()
        }
        return nil
    })
    
    http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
        if req.Method == "POST" {
            log.Info("it`s POST!")
        }
        fmt.Fprint(res, "Hello: "+req.Host)
    })
    
    group.Go(func() error {
        log.Infof("Starting web server on %s", conf.HTTPListenIPPort)
        log.Infof("server: %v", server)
        err := server.ListenAndServe()
        return err
    })

    err := group.Wait()
    if err != nil {
        if errors.Is(err, context.Canceled) {
            log.Error("Context was canceled")
        } else {
            log.Errorf("Received error: %v\n", err)
        }
    } else {
        log.Error("Sucsessfull finished")
    }
}
