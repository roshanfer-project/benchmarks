package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hotel"
	dagorinit "hotel/dagor_init"
	rajomoninit "hotel/rajomon_init"
	"hotel/utils"
	"net/http"
	"strconv"

	oteltool "hotel/otel_tool"
	pb "hotel/protobuf"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "client"

var log = utils.GetLogger(serviceName)
var tracer trace.Tracer
var dagor bool
var rajomon bool

type Server struct {
	frontendClient pb.FrontendServiceClient
}

func tracingMiddleware1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func configOTL(ctx context.Context, serviceName string) (grpc.ServerOption, []func(context.Context) error, bool) {
	if shutdownList, ok := oteltool.InitializeOTel(ctx, serviceName, false); ok {
		//tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")
		//meter = otel.GetMeterProvider().Meter(serviceName + "-meter")
		return opentelemetry.ServerOption(opentelemetry.Options{
			MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: otel.GetMeterProvider()}}), shutdownList, true
	} else {
		return nil, nil, false
	}

}

func (s *Server) Run() error {

	log.Info("Initializing client...")
	var tracingMiddleware func(next http.Handler) http.Handler
	tracingMiddleware = tracingMiddleware1

	rajomon = utils.GetEnvVar("rajomon", false) == "true"
	dagor = utils.GetEnvVar("dagor", false) == "true"
	if !rajomon && !dagor {
		panic("Either Rajomon or Dagor must be enabled")
	}

	log.Info("Initializing gRPC clients...")

	frontendEnv := utils.GetEnvVar("FrontendGRPCAddr", true)
	var conn *grpc.ClientConn

	if rajomon {
		log.Info("Rajomon is enabled, initializing Rajomon client...")
		priceTable := rajomoninit.GetPriceTable(serviceName, true)
		conn = hotel.GetRajomonClient(frontendEnv, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorEnduser))
	} else if dagor {
		log.Info("Dagor is enabled, initializing Dagor client...")
		dagorNode := dagorinit.GetDagorNode(serviceName, false, true)
		conn = hotel.GetConn(frontendEnv, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	s.frontendClient = pb.NewFrontendServiceClient(conn)
	mux := http.NewServeMux()
	mux.Handle("/hotels", tracingMiddleware(http.HandlerFunc(s.searchHandler)))
	mux.Handle("/reservation", tracingMiddleware(http.HandlerFunc(s.reservationHandler)))
	tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")

	log.Info("Successful")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("ClientPort", true))),
		Handler: mux,
	}

	log.Info("Serving http")
	return srv.ListenAndServe()
}

func main() {
	src := &Server{}
	if err := src.Run(); err != nil {
		log.Error("Failed to run server: " + err.Error())
		panic(err)
	}
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("starts searchHandler")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "search-hotel")
	arg := &pb.SearchHotelsRequest{
		InDate:  r.URL.Query().Get("inDate"),
		OutDate: r.URL.Query().Get("outDate"),
		Lat:     float32(utils.ParseFloatString(r.URL.Query().Get("lat"))),
		Lon:     float32(utils.ParseFloatString(r.URL.Query().Get("lon"))),
	}
	resp, err := s.frontendClient.SearchHotels(ctx, arg)
	if err != nil {
		log.Error("SearchHotels RPC failed: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := geoJSONResponse(resp.Profiles)
	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// disable chunked encoding by setting Content-Length header
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

func (s *Server) reservationHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("starts reservationHandler")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "reserve-hotel")
	arg := &pb.FrontendReservationRequest{
		HotelId:      r.URL.Query().Get("hotelId"),
		InDate:       r.URL.Query().Get("inDate"),
		OutDate:      r.URL.Query().Get("outDate"),
		Number:       int32(utils.StrToInt(r.URL.Query().Get("number"))),
		CustomerName: r.URL.Query().Get("customerName"),
		Username:     r.URL.Query().Get("username"),
		Password:     r.URL.Query().Get("password"),
	}
	resp, err := s.frontendClient.FrontendReservation(ctx, arg)
	if err != nil {
		log.Error("FrontendReservation RPC failed: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	responseBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// disable chunked encoding by setting Content-Length header
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

func geoJSONResponse(hs []*pb.Hotel) map[string]interface{} {
	fs := []interface{}{}

	for _, h := range hs {
		fs = append(fs, map[string]interface{}{
			"type": "Feature",
			"id":   h.Id,
			/* "properties": map[string]string{
				"name":         h.Name,
				"phone_number": h.PhoneNumber,
			},
			"geometry": map[string]interface{}{
				"type": "Point",
				"coordinates": []float32{
					h.Address.Lon,
					h.Address.Lat,
				},
			}, */
		})
	}

	return map[string]interface{}{
		"type":     "FeatureCollection",
		"features": fs,
	}
}
