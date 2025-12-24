package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"hotel"
	profile "hotel/protobuf"
	reservation "hotel/reservation/proto"
	search "hotel/search/proto"
	user "hotel/user/proto"
	"hotel/utils"
	"io/fs"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

var (
	//go:embed static/*
	content embed.FS
)

type Server struct {
	searchClient  search.SearchClient
	profileClient profile.ProfileClient
	//recommendationClient recommendation.RecommendationClient
	userClient user.UserClient
	//reviewClient         review.ReviewClient
	//attractionsClient    attractions.AttractionsClient
	reservationClient reservation.ReservationClient
}

const serviceName = "frontend"

var log = utils.GetLogger(serviceName)
var tracer trace.Tracer
var sidecar bool

type CounterState struct {
	failedRPCCounter atomic.Int64
	inReq            sync.Map
	outReq           sync.Map
	maxQueue         sync.Map
	lock             sync.Mutex
}

func (s *CounterState) IncrementInReq(method string) {
	count, _ := s.inReq.LoadOrStore(method, int64(0))
	s.inReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementOutReq(method string) {
	count, _ := s.outReq.LoadOrStore(method, int64(0))
	s.outReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementMaxQueue(method string, value int64) {
	count, _ := s.maxQueue.LoadOrStore(method, int64(0))
	if value > count.(int64) {
		s.maxQueue.Store(method, value)
	}
}

func (s *CounterState) GetMaxQueue(method string) int64 {
	count, ok := s.maxQueue.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetFailedRPCCounter() int64 {
	return s.failedRPCCounter.Load()
}

func (s *CounterState) IncrementFailedRPCCounter() {
	s.failedRPCCounter.Add(1)
}

func (s *CounterState) GetInReq(method string) int64 {
	count, ok := s.inReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetOutReq(method string) int64 {
	count, ok := s.outReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

var maxQueueGuage metric.Int64Gauge
var standardMethods map[string]string
var counters = &CounterState{}

func tracingMiddleware1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tracingMiddleware wraps an http.Handler and starts a trace span for each request.
func tracingMiddleware2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		// extract first part of the path as method name
		method := r.URL.Path[1:]
		defer span.End()
		counters.lock.Lock()
		counters.IncrementInReq(method)
		queueSize := counters.GetInReq(method) - counters.GetOutReq(method)
		if queueSize > counters.GetMaxQueue(method) {
			counters.IncrementMaxQueue(method, queueSize)
			maxQueueGuage.Record(ctx, queueSize, metric.WithAttributes(attribute.String("api", standardMethods[method])))
		}
		counters.lock.Unlock()
		next.ServeHTTP(w, r.WithContext(ctx))
		counters.lock.Lock()
		counters.IncrementOutReq(method)
		counters.lock.Unlock()
	})
}

func (s *Server) Run() error {
	var tracingMiddleware func(next http.Handler) http.Handler
	if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") ||
		(utils.GetEnvVar("plain", false) == "true") {
		tracingMiddleware = tracingMiddleware1
	} else {
		tracingMiddleware = tracingMiddleware1
	}

	sidecar = utils.GetEnvVar("sidecar", false) == "true"

	log.Info("Loading static content...")
	staticContent, err := fs.Sub(content, "static")
	if err != nil {
		return err
	}

	log.Info("Initializing gRPC clients...")
	var searchEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		searchEnv = "FrontendEgress"
	} else {
		searchEnv = "SearchAddr"
	}
	conn := hotel.GetConn(utils.GetEnvVar(searchEnv, true))
	s.searchClient = search.NewSearchClient(conn)

	var profileEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		profileEnv = "FrontendEgress"
	} else {
		profileEnv = "ProfileAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(profileEnv, true))
	s.profileClient = profile.NewProfileClient(conn)

	/* if err := s.initRecommendationClient("srv-recommendation"); err != nil {
		return err
	} */
	var userEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		userEnv = "FrontendEgress"
	} else {
		userEnv = "UserAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(userEnv, true))
	s.userClient = user.NewUserClient(conn)

	var reservationEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		reservationEnv = "FrontendEgress"
	} else {
		reservationEnv = "ReservationAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(reservationEnv, true))
	s.reservationClient = reservation.NewReservationClient(conn)

	/* if err := s.initReviewClient("srv-review"); err != nil {
		return err
	}

	if err := s.initAttractionsClient("srv-attractions"); err != nil {
		return err
	} */

	log.Info("Successful")

	// Configure OTL
	/* ctx := context.Background()
	hotel.ConfigOTL(ctx, serviceName, true) */
	tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")

	//log.Trace().Msg("frontend before mux")
	//mux := tracing.NewServeMux(s.Tracer)
	mux := http.NewServeMux()
	mux.Handle("/", tracingMiddleware((http.FileServer(http.FS(staticContent)))))
	mux.Handle("/hotels", tracingMiddleware(http.HandlerFunc(s.searchHandler)))
	/* mux.Handle("/recommendations", http.HandlerFunc(s.recommendHandler))
	mux.Handle("/user", http.HandlerFunc(s.userHandler))
	mux.Handle("/review", http.HandlerFunc(s.reviewHandler))
	mux.Handle("/restaurants", http.HandlerFunc(s.restaurantHandler))
	mux.Handle("/museums", http.HandlerFunc(s.museumHandler))
	mux.Handle("/cinema", http.HandlerFunc(s.cinemaHandler)) */
	mux.Handle("/reservation", tracingMiddleware(http.HandlerFunc(s.reservationHandler)))

	log.Info("frontend starts serving")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("FrontendPort", true))),
		Handler: mux,
	}

	log.Info("Serving http")
	return srv.ListenAndServe()
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var rpc_id string
	if sidecar {
		rpc_id = r.Header.Get("rpc-id")
		if rpc_id == "" {
			http.Error(w, "Please specify rpc-id", http.StatusBadRequest)
			return
		}
	} else {
		rpc_id = ""
	}
	ctx := r.Context()
	// add method to context
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "search-hotel", "rpc-id", rpc_id))

	log.Debug("starts searchHandler")

	// in/out dates from query params
	inDate, outDate := r.URL.Query().Get("inDate"), r.URL.Query().Get("outDate")
	if inDate == "" || outDate == "" {
		http.Error(w, "Please specify inDate/outDate params", http.StatusBadRequest)
		return
	}

	// lan/lon from query params
	sLat, sLon := r.URL.Query().Get("lat"), r.URL.Query().Get("lon")
	if sLat == "" || sLon == "" {
		http.Error(w, "Please specify location params", http.StatusBadRequest)
		return
	}

	Lat, _ := strconv.ParseFloat(sLat, 32)
	lat := float32(Lat)
	Lon, _ := strconv.ParseFloat(sLon, 32)
	lon := float32(Lon)

	log.Debug("starts searchHandler querying downstream")

	log.Debug(fmt.Sprintf("SEARCH [lat: %v, lon: %v, inDate: %v, outDate: %v", lat, lon, inDate, outDate))
	// search for best hotels
	searchResp, err := s.searchClient.Nearby(ctx, &search.NearbyRequest{
		Lat:     lat,
		Lon:     lon,
		InDate:  inDate,
		OutDate: outDate,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("SearchHandler gets searchResp")
	//for _, hid := range searchResp.HotelIds {
	//	log.Trace().Msgf("Search Handler hotelId = %s", hid)
	//}

	// grab locale from query params or default to en
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "en"
	}

	reservationResp, err := s.reservationClient.CheckAvailability(ctx, &reservation.Request{
		CustomerName: "",
		HotelId:      searchResp.HotelIds,
		InDate:       inDate,
		OutDate:      outDate,
		RoomNumber:   1,
	})
	if err != nil {
		log.Error("SearchHandler CheckAvailability failed " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("searchHandler gets reserveResp")
	log.Debug(fmt.Sprintf("searchHandler gets reserveResp.HotelId = %s", reservationResp.HotelId))

	// hotel profiles
	profileResp, err := s.profileClient.GetProfiles(ctx, &profile.Request{
		HotelIds: reservationResp.HotelId,
		Locale:   locale,
	})
	if err != nil {
		log.Error("SearchHandler GetProfiles failed " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("searchHandler gets profileResp")

	// generate the geoJSON response
	response := geoJSONResponse(profileResp.Hotels)
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx := r.Context()
	var rpc_id string
	if sidecar {
		rpc_id = r.Header.Get("rpc-id")
		if rpc_id == "" {
			http.Error(w, "Please specify rpc-id", http.StatusBadRequest)
			return
		}
	} else {
		rpc_id = ""
	}
	// add method to context
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "reserve-hotel", "rpc-id", rpc_id))

	inDate, outDate := r.URL.Query().Get("inDate"), r.URL.Query().Get("outDate")
	if inDate == "" || outDate == "" {
		http.Error(w, "Please specify inDate/outDate params", http.StatusBadRequest)
		return
	}

	if !checkDataFormat(inDate) || !checkDataFormat(outDate) {
		http.Error(w, "Please check inDate/outDate format (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	hotelId := r.URL.Query().Get("hotelId")
	if hotelId == "" {
		http.Error(w, "Please specify hotelId params", http.StatusBadRequest)
		return
	}

	customerName := r.URL.Query().Get("customerName")
	if customerName == "" {
		http.Error(w, "Please specify customerName params", http.StatusBadRequest)
		return
	}

	username, password := r.URL.Query().Get("username"), r.URL.Query().Get("password")
	if username == "" || password == "" {
		http.Error(w, "Please specify username and password", http.StatusBadRequest)
		return
	}

	numberOfRoom := 0
	num := r.URL.Query().Get("number")
	if num != "" {
		numberOfRoom, _ = strconv.Atoi(num)
	}

	// Check username and password
	recResp, err := s.userClient.CheckUser(ctx, &user.Request{
		Username: username,
		Password: password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	str := "Reserve successfully!"
	if !recResp.Correct {
		str = "Failed. Please check your username and password. "
	}

	// Make reservation
	resResp, err := s.reservationClient.MakeReservation(ctx, &reservation.Request{
		CustomerName: customerName,
		HotelId:      []string{hotelId},
		InDate:       inDate,
		OutDate:      outDate,
		RoomNumber:   int32(numberOfRoom),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(resResp.HotelId) == 0 {
		str = "Failed. Already reserved. "
	}

	res := map[string]interface{}{
		"message": str,
	}

	responseBytes, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

// return a geoJSON response that allows google map to plot points directly on map
// https://developers.google.com/maps/documentation/javascript/datalayer#sample_geojson
func geoJSONResponse(hs []*profile.Hotel) map[string]interface{} {
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

func main() {
	standardMethods = make(map[string]string)
	standardMethods["hotels"] = "search-hotel"
	standardMethods["reservation"] = "reserve-hotel"
	/* var ok error
	maxQueueGuage, ok = otel.GetMeterProvider().Meter(serviceName).Int64Gauge("max_queue",
		metric.WithDescription("Maximum queue length for each RPC method"))
	if ok != nil {
		log.Error("Failed to create max_queue gauge")
		panic("Failed to create max_queue gauge")
	} */
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start server", "error", err)
	}
	log.Info("Server stopped")
}

func checkDataFormat(date string) bool {
	if len(date) != 10 {
		return false
	}
	for i := 0; i < 10; i++ {
		if i == 4 || i == 7 {
			if date[i] != '-' {
				return false
			}
		} else {
			if date[i] < '0' || date[i] > '9' {
				return false
			}
		}
	}
	return true
}
