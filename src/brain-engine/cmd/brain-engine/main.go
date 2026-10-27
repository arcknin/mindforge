package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/api"
	"github.com/arcknin/mindforge/src/brain-engine/auth"
	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/documents"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, os.Getenv, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(ctx context.Context, getenv func(string) string, output io.Writer) error {
	config, err := loadRuntimeConfig(getenv)
	if err != nil {
		return err
	}
	// Do not forward authenticated internal requests to an ambient proxy.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	defer transport.CloseIdleConnections()
	upstreamURL := strings.TrimSuffix(config.authUserURL, "/user")
	facade, err := auth.NewFacade(auth.FacadeConfig{
		UpstreamURL:       upstreamURL,
		CallbackURL:       config.authCallbackURL,
		RedirectAllowlist: config.authRedirectURLs,
	}, &http.Client{Transport: transport})
	if err != nil {
		return errors.New("develop Auth facade configuration failed")
	}
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	store, err := initializeStore(startup, config, probeDatabase, func(c context.Context, dsn string) (runtimeStore, error) { return brain.OpenPostgresStore(c, dsn) })
	cancel()
	if err != nil {
		return err
	}
	defer store.Close()
	documentStartup, documentCancel := context.WithTimeout(ctx, 10*time.Second)
	documentStore, err := documents.OpenPostgresStore(documentStartup, config.databaseURL)
	documentCancel()
	if err != nil {
		return errors.New("develop document persistence store could not be opened")
	}
	defer documentStore.Close()
	listener, err := net.Listen("tcp", config.listenAddress)
	if err != nil {
		return errors.New("develop API listener could not be opened")
	}
	defer listener.Close()
	server := &http.Server{
		Handler:           api.NewHandlerWithAuth(brain.NewEngine(store, brain.NewUUIDSource()), documents.NewEngine(documentStore), facade, facade),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10,
		// net/http diagnostics must not include request paths, credentials or peers.
		ErrorLog: log.New(io.Discard, "", 0),
	}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if server.Shutdown(shutdown) != nil {
				_ = server.Close()
			}
		case <-stopped:
		}
	}()
	fmt.Fprintln(output, "Brain Engine develop: listener started")
	err = server.Serve(listener)
	close(stopped)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("develop API server stopped unexpectedly")
	}
	return nil
}
