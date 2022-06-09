package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type application struct {
	config    *configuration
	dnsClient *dnsClient
	timeout   time.Duration
	logger    *logrus.Logger
}

func main() {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	err := godotenv.Load()
	if err != nil {
		log.Warnf("Error loading .env file: %v. continuing without", err)
	}

	host := flag.String("host", lookupEnvOrString(log, "NGINX_HOST", "127.0.0.1:8080"), "IP and Port to bind to. You can also use the NGINX_HOST environment variable or an entry in the .env file to set this parameter.")
	debug := flag.Bool("debug", lookupEnvOrBool(log, "NGINX_DEBUG", false), "Enable DEBUG mode. You can also use the NGINX_DEBUG environment variable or an entry in the .env file to set this parameter.")
	wait := flag.Duration("graceful-timeout", lookupEnvOrDuration(log, "NGINX_GRACEFUL_TIMEOUT", 2*time.Second), "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m. You can also use the NGINX_GRACEFUL_TIMEOUT environment variable or an entry in the .env file to set this parameter.")
	timeout := flag.Duration("timeout", lookupEnvOrDuration(log, "NGINX_TIMEOUT", 5*time.Second), "dns and http timeout. You can also use the NGINX_TIMEOUT environment variable or an entry in the .env file to set this parameter.")
	configFile := flag.String("config", lookupEnvOrString(log, "NGINX_HOST", "config.json"), "config file to use. You can also use the NGINX_HOST environment variable or an entry in the .env file to set this parameter.")
	flag.Parse()

	if *debug {
		log.SetLevel(logrus.DebugLevel)
		log.Debug("DEBUG mode enabled")
	}

	config, err := getConfig(*configFile)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	dnsClient := newDNSClient(*timeout)

	app := &application{
		config:    config,
		dnsClient: dnsClient,
		timeout:   *timeout,
		logger:    log,
	}

	srv := &http.Server{
		Addr:    *host,
		Handler: app.routes(),
	}
	log.Infof("Starting server on %s", *host)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), *wait)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(err)
	}
	log.Info("shutting down")
	os.Exit(0)
}

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(app.timeout))

	h := http.HandlerFunc(app.authHandler)
	r.Handle("/auth", h)
	return r
}

func (app *application) logError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Connection", "close")
	errorText := fmt.Sprintf("%v", err)
	app.logger.Error(errorText)
	http.Error(w, http.StatusText(status), status)
}

func (app *application) authHandler(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		app.logError(w, err, http.StatusBadRequest)
		return
	}

	for _, d := range app.config.Domains {
		dynamicIP, err := app.dnsClient.ipLookup(r.Context(), d)
		if err != nil {
			app.logError(w, fmt.Errorf("invalid domain %s in config: %w", d, err), http.StatusInternalServerError)
			return
		}
		for _, i := range dynamicIP {
			if i == ip {
				app.logger.Infof("allowing client %s with hostnames %s", ip, d)
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	app.logger.Infof("denying client %s", ip)
	w.WriteHeader(http.StatusUnauthorized)
}
