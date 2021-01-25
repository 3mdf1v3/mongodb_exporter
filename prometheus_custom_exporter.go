package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoDbHostURI string
var mongoDbAuthSource string
var mongoDbUsername string
var mongoDbPassword string
var (
	asocesip = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asoc_es_ip",
			Help: "",
		},
		[]string{"_id"},
	)
)

func init() {
	prometheus.MustRegister(asocesip)
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var clientOptions *options.ClientOptions
		if mongoDbAuthSource == "" && mongoDbUsername == "" && mongoDbPassword == "" {
			clientOptions = options.Client().ApplyURI(mongoDbHostURI)
		} else {
			clientOptions = options.Client().ApplyURI(mongoDbHostURI).
				SetAuth(options.Credential{
					AuthSource: mongoDbAuthSource, Username: mongoDbUsername, Password: mongoDbPassword,
				})
		}

		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			log.Fatal(err)
		}

		err = client.Ping(context.TODO(), nil)
		if err != nil {
			log.Fatal(err)
		}

		eventsourcesCollection := client.Database("esm").Collection("eventsources")

		findOptions := options.Find()
		findOptions.SetLimit(100)
		cur, err := eventsourcesCollection.Find(context.TODO(), bson.D{{}}, findOptions)
		if err != nil {
			log.Fatal(err)
		}

		for cur.Next(context.TODO()) {

			var result map[string]interface{}
			err := cur.Decode(&result)
			if err != nil {
				log.Fatal(err)
			}

			var attributes map[string]interface{}
			attributes = result["attributes"].(map[string]interface{})

			var asoceslastSeen float64
			asoceslastSeen = float64(attributes["asoc-es-lastSeen"].(int64))
			asocesip.WithLabelValues(result["_id"].(string)).Add(asoceslastSeen)
		}

		next.ServeHTTP(w, r)
		asocesip.Reset()
	})
}

func main() {

	var localPort string

	flag.StringVar(&mongoDbHostURI, "mongoDbHostURI", "localhost:27017", "MongoDB uri default localhost:27017")
	flag.StringVar(&mongoDbAuthSource, "mongoDbAuthSource", "", "MongoDB authentication source")
	flag.StringVar(&mongoDbUsername, "mongoDbUsername", "", "MongoDB authentication username")
	flag.StringVar(&mongoDbPassword, "mongoDbPassword", "", "MongoDB authentication password")
	flag.StringVar(&localPort, "localPort", "15700", "Exporter listening uri default 8080")

	flag.Parse()

	localPort = ":" + localPort
	mongoDbHostURI = "mongodb://" + mongoDbHostURI

	mux := mux.NewRouter()
	mux.Use(prometheusMiddleware)
	mux.Path("/metrics").Handler(promhttp.Handler())

	fmt.Printf("Using MongoDB server: %v\n", mongoDbHostURI)
	fmt.Printf("Starting server on port: %v\n", localPort)
	log.Fatal(http.ListenAndServe(localPort, mux))

}
