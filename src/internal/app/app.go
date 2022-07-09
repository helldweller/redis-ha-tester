package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var ctx, cancel = context.WithCancel(context.Background())
var group, groupCtx = errgroup.WithContext(ctx)
var server *http.Server
var expiration = time.Second * 300

// Run is a like main function
func Run() {
	log.Info("Starting app")

	server = &http.Server{
		Addr:    conf.HTTPListenIPPort,
		Handler: nil,
		// BaseContext: ctx,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

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
		b := md5.Sum([]byte(req.RequestURI))
		key := hex.EncodeToString(b[:])
		if req.Method == "POST" {
			req.Body = http.MaxBytesReader(res, req.Body, 512) // default redis value max size
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				res.WriteHeader(http.StatusRequestEntityTooLarge)
				log.Warnf("%s, %s, %s, %s", req.Method, req.Host, req.RequestURI, err)
				return
			} else {
				err = rdb.Set(ctx, key, body, expiration).Err()
				if err != nil {
					res.WriteHeader(http.StatusInternalServerError)
					log.Errorf("Redis key setting error. %s, key: %s", err, key)
					return
				}
				fmt.Fprint(res, "OK")
			}
		}
		if req.Method == "GET" {
			val, err := rdb.Get(ctx, key).Result()
			if err == redisNil {
				res.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				log.Errorf("Redis key getting error. %s, key: %s", err, key)
			}
			fmt.Fprint(res, val)
		}
	})

	group.Go(func() error {
		log.Infof("Starting web server on %s", conf.HTTPListenIPPort)
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
