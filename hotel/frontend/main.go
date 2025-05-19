package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"hotel"
	profile "hotel/profile/proto"
	reservation "hotel/reservation/proto"
	search "hotel/search/proto"
	"hotel/utils"
	"io/fs"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	//go:embed static/*
	content embed.FS
)

type Server struct {
	searchClient  search.SearchClient
	profileClient profile.ProfileClient
	//recommendationClient recommendation.RecommendationClient
	//userClient           user.UserClient
	//reviewClient         review.ReviewClient
	//attractionsClient    attractions.AttractionsClient
	reservationClient reservation.ReservationClient
}

var log = utils.GetLogger("frontend")
var tracer trace.Tracer

// tracingMiddleware wraps an http.Handler and starts a trace span for each request.
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) Run() error {

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
	}

	if err := s.initUserClient("srv-user"); err != nil {
		return err
	} */

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
	ctx := context.Background()
	hotel.ConfigOTL(ctx, "frontend", true)
	tracer = otel.GetTracerProvider().Tracer("frontend-tracer")

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
	mux.Handle("/cinema", http.HandlerFunc(s.cinemaHandler))
	mux.Handle("/reservation", http.HandlerFunc(s.reservationHandler)) */

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
	ctx := r.Context()

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

	json.NewEncoder(w).Encode(geoJSONResponse(profileResp.Hotels))
}

// return a geoJSON response that allows google map to plot points directly on map
// https://developers.google.com/maps/documentation/javascript/datalayer#sample_geojson
func geoJSONResponse(hs []*profile.Hotel) map[string]interface{} {
	fs := []interface{}{}

	for _, h := range hs {
		fs = append(fs, map[string]interface{}{
			"type": "Feature",
			"id":   h.Id,
			"properties": map[string]string{
				"name":         h.Name,
				"phone_number": h.PhoneNumber,
			},
			"geometry": map[string]interface{}{
				"type": "Point",
				"coordinates": []float32{
					h.Address.Lon,
					h.Address.Lat,
				},
			},
		})
	}

	return map[string]interface{}{
		"type":     "FeatureCollection",
		"features": fs,
	}
}

func main() {
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start server", "error", err)
	}
	log.Info("Server stopped")
}
