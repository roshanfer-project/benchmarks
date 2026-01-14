// This is a simple http server

package main

import (
	"context"
	"fmt"
	"net/http"

	_ "net/http/pprof"
	"strconv"
	"strings"
	"test1/utils"

	"google.golang.org/grpc/metadata"
)

var deployment string
var appSize int
var app2Size int
var app3Size int
var lastRpcId int

var appPreRepeat int
var appPostRepeat int

var app2PreRepeat int
var app2PostRepeat int

var app3PreRepeat int
var app3PostRepeat int

var listenPort int
var sidecar bool

var log = utils.GetLogger("app")

func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := getContextWithRpcId(r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	mux := http.NewServeMux()
	/* http.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		appLogic(w, getContextWithRpcId(r))
	}) */
	mux.Handle("/app", tracingMiddleware(http.HandlerFunc(appLogic)))
	mux.Handle("/app2", tracingMiddleware(http.HandlerFunc(app2Logic)))
	mux.Handle("/app3", tracingMiddleware(http.HandlerFunc(app3Logic)))

	// Start pprof server
	go func() {
		pprofPort := 6000
		log.Info("Starting pprof server", "port", pprofPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil); err != nil {
			log.Error("pprof server failed", "error", err)
		}
	}()

	/* http.HandleFunc("/app2", func(w http.ResponseWriter, r *http.Request) {
		app2Logic(w, getContextWithRpcId(r))
	}) */
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: mux,
	}
	log.Info("Starting server on port", "listenPort", listenPort)
	srv.ListenAndServe()
}

func getContextWithRpcId(r *http.Request) context.Context {
	var rpcId string
	var api string
	if sidecar {
		rpcId = r.Header.Get("rpc-id")
		api = r.Pattern[1:]
		/* if rpcId == "" {
			log.Error("rpc-id header is required")
			return r.Context()
		} */
	} else {
		rpcId = strconv.Itoa(lastRpcId)
		r.Header.Set("rpc-id", rpcId)
		lastRpcId++
		api = r.Pattern[1:]
	}
	md := metadata.New(map[string]string{"rpc-id": rpcId, "api": api})
	return metadata.NewOutgoingContext(r.Context(), md)
}

func init() {
	lastRpcId = 1
	deployment = utils.GetEnvVar("deployment", true)
	sidecar = utils.GetEnvVar("sidecar", true) == "true"
	listenPort = utils.StrToInt(utils.GetEnvVar("appListenPort", true))
	appSize = utils.StrToInt(utils.GetEnvVar("appSize", true))
	app2Size = utils.StrToInt(utils.GetEnvVar("app2Size", true))
	appPreRepeat = utils.StrToInt(utils.GetEnvVar("appPreRepeat", true))
	appPostRepeat = utils.StrToInt(utils.GetEnvVar("appPostRepeat", true))
	app2PreRepeat = utils.StrToInt(utils.GetEnvVar("app2PreRepeat", true))
	app2PostRepeat = utils.StrToInt(utils.GetEnvVar("app2PostRepeat", true))
	app3PreRepeat = utils.StrToInt(utils.GetEnvVar("app3PreRepeat", true))
	app3PostRepeat = utils.StrToInt(utils.GetEnvVar("app3PostRepeat", true))
	app3Size = utils.StrToInt(utils.GetEnvVar("app3Size", true))
	fmt.Printf("deployment: %s\n", deployment)
	fmt.Printf("appSize: %d\n", appSize)
}

func makebigString(size int) string {
	return strings.Repeat("a", size)
}

func busyLoop(repeat int) {
	for range repeat {
		for range 10000 {
		}
	}
}

func writeResponseWithoutchunkEncoding(w http.ResponseWriter, data string) {
	// convert to bytes
	responseBytes := []byte(data)
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	// write the data
	w.Write(responseBytes)

}

// with no chunk-encoding
func appLogic(w http.ResponseWriter, r *http.Request) {
	switch deployment {
	case "test1":
		busyLoop(appPreRepeat + appPostRepeat)
		writeResponseWithoutchunkEncoding(w, makebigString(appSize))
	default:
		panic("Unknown deployment")
	}
}

func app2Logic(w http.ResponseWriter, r *http.Request) {
	switch deployment {
	case "test1":
		busyLoop(app2PreRepeat + app2PostRepeat)
		writeResponseWithoutchunkEncoding(w, makebigString(app2Size))
	default:
		panic("Unknown deployment")
	}
}

func app3Logic(w http.ResponseWriter, r *http.Request) {
	switch deployment {
	case "test1":
		busyLoop(app3PreRepeat + app3PostRepeat)
		writeResponseWithoutchunkEncoding(w, makebigString(app3Size))
	default:
		panic("Unknown deployment")
	}
}
