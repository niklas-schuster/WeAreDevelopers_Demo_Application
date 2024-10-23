package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
)

type TodoItem struct {
	ID   string `json:"id"`
	Task string `json:"task"`
}

var bucketName string
var client *storage.Client
var ctx context.Context

func init() {
	bucketName = os.Getenv("GCP_BUCKET_NAME")
	ctx = context.Background()

	var err error
	client, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	bucket := client.Bucket(bucketName)
	query := &storage.Query{}
	it := bucket.Objects(ctx, query)

	var items []TodoItem
	for {
		objAttrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			log.Printf("Error listing objects: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rc, err := bucket.Object(objAttrs.Name).NewReader(ctx)
		if err != nil {
			log.Printf("Error reading object: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Printf("Error reading object data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var item TodoItem
		if err := json.Unmarshal(data, &item); err != nil {
			log.Printf("Error unmarshalling object data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		items = append(items, item)
	}

	json.NewEncoder(w).Encode(items)
}

func addTodo(w http.ResponseWriter, r *http.Request) {
	var item TodoItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure ID is provided
	if item.ID == "" {
		log.Println("Missing ID in the request body")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	bucket := client.Bucket(bucketName)
	obj := bucket.Object(item.ID).NewWriter(ctx)
	defer obj.Close()

	data, err := json.Marshal(item)
	if err != nil {
		log.Printf("Error marshalling item: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := obj.Write(data); err != nil {
		log.Printf("Error writing to bucket: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	todoID := vars["id"]

	// Ensure ID is provided
	if todoID == "" {
		log.Println("Missing ID in the request URL")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	bucket := client.Bucket(bucketName)
	obj := bucket.Object(todoID)

	// Delete the object from the bucket
	if err := obj.Delete(ctx); err != nil {
		log.Printf("Error deleting object: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/todos", getTodos).Methods("GET")
	r.HandleFunc("/todos", addTodo).Methods("POST")
	r.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE") // New DELETE route

	fmt.Println("Starting server on :9090")
	log.Fatal(http.ListenAndServe(":9090", r))
}
