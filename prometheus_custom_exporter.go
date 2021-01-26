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

var mongoDbHostURI, mongoDbAuthSource, mongoDbUsername, mongoDbPassword string
var (
	esmMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asoc_es_ip",
			Help: "MongoDB esm database export",
		},
		[]string{"asocestype", "asocesip", "asocesaddress", "asoceslogCollector", "asoceslogDecoder"},
	)
)

func init() {
	prometheus.MustRegister(esmMetrics)
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
			var asocestype, asocesip, asocesaddress, asoceslogCollector, asoceslogDecoder string

			if attributes["asoc-es-lastSeen"] != nil {
				asoceslastSeen = float64(attributes["asoc-es-lastSeen"].(int64))
			}
			if attributes["asoc-es-type"] != nil {
				asocestype = attributes["asoc-es-type"].(string)
			}
			if attributes["asoc-es-ip"] != nil {
				asocesip = attributes["asoc-es-ip"].(string)
			}
			if attributes["asoc-es-address"] != nil {
				asocesaddress = attributes["asoc-es-address"].(string)
			}
			if attributes["asoc-es-logCollector"] != nil {
				asoceslogCollector = attributes["asoc-es-logCollector"].(string)
			}
			if attributes["asoc-es-logDecoder"] != nil {
				asoceslogDecoder = attributes["asoc-es-logDecoder"].(string)
			}

			esmMetrics.WithLabelValues(asocestype, asocesip, asocesaddress, asoceslogCollector, asoceslogDecoder).Add(asoceslastSeen)
		}

		next.ServeHTTP(w, r)
		esmMetrics.Reset()
		client.Disconnect(context.TODO())
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
