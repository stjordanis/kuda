package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cyrildiagne/kuda/pkg/api"
	"github.com/cyrildiagne/kuda/pkg/deploy"
	"github.com/cyrildiagne/kuda/pkg/gcloud"

	"github.com/gorilla/mux"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello!\n")
}

func main() {
	gcpProjectID := os.Getenv("KUDA_GCP_PROJECT")
	if gcpProjectID == "" {
		panic("cloud not load env var KUDA_GCP_PROJECT")
	}
	log.Println("Using project:", gcpProjectID)

	if err := gcloud.AuthServiceAccount(); err != nil {
		log.Fatalf("error authenticating with credentials. %v\n", err)
	}

	if err := gcloud.GetKubeConfig(gcpProjectID); err != nil {
		log.Fatalf("could not retrieve kubectl credentials %v\n", err)
	}

	auth, fs, err := gcloud.InitFirebase(gcpProjectID)
	if err != nil {
		log.Fatalf("error initializing firebase: %v\n", err)
	}

	env := &api.Env{
		GCPProjectID: gcpProjectID,
		DB:           fs,
		Auth:         auth,
	}

	// user := "cyrildiagne"
	// api := "hello-gpu"
	// image := env.GetDockerImagePath(user, api)
	// if err := gcloud.ListImageTags(image); err != nil {
	// 	panic(err)
	// }

	port := "8080"
	if value, ok := os.LookupEnv("port"); ok {
		port = value
	}
	fmt.Println("Starting api on port", port)

	r := mux.NewRouter()
	r.HandleFunc("/", handleRoot).Methods("GET")

	deployHandler := api.Handler{Env: env, H: deploy.HandleDeploy}
	r.Handle("/deploy", deployHandler).Methods("POST")

	publishHandler := api.Handler{Env: env, H: deploy.HandlePublish}
	r.Handle("/publish", publishHandler).Methods("POST")

	http.ListenAndServe(":"+port, r)
}