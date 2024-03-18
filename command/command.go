package command

import (
	ssl "crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	cfg "github.com/uninus-opensource/uninus-go-architect-common/config"
	run "github.com/uninus-opensource/uninus-go-architect-common/grcp"

	logkit "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/gorilla/handlers"
	"github.com/opentracing/opentracing-go"
	"github.com/uninus-opensource/uninus-go-architect-common/log"
	util "github.com/uninus-opensource/uninus-go-architect-common/microservice"
	"google.golang.org/grpc/credentials"
)

type ServerConfig struct {
	DiscoveredHosts                      []string
	IP, Port, AllowedOrigins, TracerHost string
	CAPath, CertPath, PrivateKey         string
	ConfigFlag                           *string
	IPFlag                               *string
	PortFlag                             *string
}

// ServerCommandConf this struct is the main body of service's server initializer
type ServerCommandConf struct {
	ServerConfig
	Logger            logkit.Logger
	ServiceID         string
	Tracers           *opentracing.Tracer
	registrar         sd.Registrar
	PreparedAddress   string
	PreparedPort      string
	PreparedIP        string
	TLS               credentials.TransportCredentials
	SSLCertificate    ssl.Certificate
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
}

type IServerCommandConf interface {
	ParseServerCommand()
	PrepareDiscoveryHostAndPort()
	ServiceRegistry()
	NilSafeRegister()
	NilSafeDeregister()
	PrepareTLSAndSSL()
	ListenAndServe() error
	InitTracerWithOpenTracing()
	DefaultHTTPHandlerWithAllowedOrigin(handler http.Handler) http.Handler
}

func NewServerCommand(
	sc ServerConfig,
	serviceID string,
	logger logkit.Logger,
	readTimeout time.Duration,
	readHeaderTimeout time.Duration,
	riteTimeout time.Duration,
) *ServerCommandConf {
	return &ServerCommandConf{
		ServerConfig:      sc,
		ServiceID:         serviceID,
		Logger:            logger,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      riteTimeout,
	}
}

// ParseServerCommand parse default commands of every service
func (scc *ServerCommandConf) ParseServerCommand() {
	flag.Parse()
	var ok bool
	if len(*scc.ConfigFlag) == 0 {
		ok = cfg.AppConfig.LoadConfig()
	} else {
		ok = cfg.AppConfig.LoadConfigFile(*scc.ConfigFlag)
	}
	if !ok {
		scc.Logger.Log(log.LogError, "failed to load configuration")
	}
}

// PrepareDiscoveryHostAndPort discover hosts and port
func (scc *ServerCommandConf) PrepareDiscoveryHostAndPort() {
	if len(*scc.IPFlag) > 0 {
		scc.IP = *scc.IPFlag
	}
	if len(*scc.PortFlag) > 0 {
		scc.Port = *scc.PortFlag
	}
	scc.PreparedAddress = fmt.Sprintf("%s:%s", scc.IP, scc.Port)
	scc.PreparedIP = scc.IP
	scc.PreparedPort = scc.Port
}

// ServiceRegistry setup registrar
func (scc *ServerCommandConf) ServiceRegistry() {
	discHost := scc.DiscoveredHosts
	address := scc.PreparedAddress
	if len(discHost) > 0 {
		var err error
		scc.registrar, err = util.ServiceRegistry(discHost, scc.ServiceID, address, scc.Logger)
		if err != nil {
			scc.Logger.Log(log.LogError, err.Error())
		}
	}
}

// NilSafeRegister register registrar with nil safe for avoid nil pointer exception
func (scc *ServerCommandConf) NilSafeRegister() {
	if scc.registrar != nil {
		scc.registrar.Register()
	}
}

// NilSafeDeregister deregister registrar with nil safe for avoid nil pointer exception
func (scc *ServerCommandConf) NilSafeDeregister() {
	if scc.registrar != nil {
		scc.registrar.Deregister()
	}
}

// PrepareTLSAndSSL prepare TLS and SSLCertificate for service server
func (scc *ServerCommandConf) PrepareTLSAndSSL() {
	var sslCert ssl.Certificate
	var tls credentials.TransportCredentials
	if len(scc.CAPath) > 0 && len(scc.CertPath) > 0 && len(scc.PrivateKey) > 0 {
		var err error
		sslCert, err = ssl.LoadX509KeyPair(scc.CertPath, scc.PrivateKey)
		if err != nil {
			scc.Logger.Log(log.LogError, err.Error())
			return
		}
		tls, err = run.TLSCredentialFromKeyPair(scc.CAPath, sslCert, true)
		if err != nil {
			scc.Logger.Log(log.LogError, err.Error())
			return
		}
	}
	scc.SSLCertificate = sslCert
	scc.TLS = tls
}

// ListenAndServe server with commons uninus configuration
func (scc *ServerCommandConf) ListenAndServe() error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", scc.PreparedPort),
		ReadTimeout:       scc.ReadTimeout,
		WriteTimeout:      scc.WriteTimeout,
		ReadHeaderTimeout: scc.ReadHeaderTimeout,
		TLSConfig: &ssl.Config{
			Certificates: []ssl.Certificate{scc.SSLCertificate},
		},
	}
	return server.ListenAndServe()
}

// DefaultHTTPHandlerWithAllowedOrigin enables cross-origin resource sharing
func (scc *ServerCommandConf) defaultHTTPHandlerWithAllowedOrigin(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// logging origin
		scc.Logger.Log(log.LogInfo, fmt.Sprintf("%+v accessor", r.Header.Get("Origin")))

		w.Header().Set("Access-Control-Allow-Origin", scc.AllowedOrigins)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
			headers := []string{"Content-Type", "Accept", "Authorization"}
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
			methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
			w.Header().Set("Access-Control-Max-Age", "86400")
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// DefaultHTTPHandler specifies default http handler
func (scc *ServerCommandConf) DefaultHTTPHandler(handler http.Handler) http.Handler {
	handler = run.LogRequestHandler(handler)
	handler = handlers.CompressHandler(handler)
	handler = scc.defaultHTTPHandlerWithAllowedOrigin(handler)
	handler = run.HealthCheckHandler(handler)
	return handler
}
