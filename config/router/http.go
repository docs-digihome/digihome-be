package router

import (
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spf13/viper"
)

func NewHTTPChi() chi.Router {
	r := chi.NewRouter()
	env := os.Getenv("ENV")
	if env != "production" {
		r.Use(middleware.Logger)
	}

	allowOrigins := viper.GetString("server.cors.allow.origins")
	allowMethods := viper.GetString("server.cors.allow.methods")
	allowHeaders := viper.GetString("server.cors.allow.headers")
	exposeHeaders := viper.GetString("server.cors.expose.headers")
	allowCredentials := viper.GetBool("server.cors.allow.credential")
	maxAge := viper.GetInt("server.cors.max.age")

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   strings.Split(allowOrigins, ","),
		AllowedMethods:   strings.Split(allowMethods, ","),
		AllowedHeaders:   strings.Split(allowHeaders, ","),
		ExposedHeaders:   strings.Split(exposeHeaders, ","),
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	}).Handler)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Minute))

	return r
}
