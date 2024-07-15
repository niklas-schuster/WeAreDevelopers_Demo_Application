package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gorilla/mux"
)

type TodoItem struct {
	ID   string `json:"id"`
	Task string `json:"task"`
}

var tableName string
var db *dynamodb.DynamoDB

func init() {
	awsRegion := os.Getenv("AWS_REGION")
	tableName = os.Getenv("DYNAMODB_TABLE_NAME")
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))
	db = dynamodb.New(sess)
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	result, err := db.Scan(&dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []TodoItem
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		log.Printf("Error unmarshalling DynamoDB items: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, "Missing ID in the request body", http.StatusBadRequest)
		return
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Printf("Error marshalling item: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		log.Printf("Error putting item into DynamoDB: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	if id == "" {
		log.Println("Missing ID in the request parameters")
		http.Error(w, "Missing ID in the request parameters", http.StatusBadRequest)
		return
	}
	fmt.Println("ID: ", id)

	// Log the ID to be deleted
	log.Printf("Deleting item with ID: %s", id)

	// Ensure the key matches the schema
	_, err := db.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	})
	if err != nil {
		log.Printf("Error deleting item from DynamoDB: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Item deleted"})
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/todos", getTodos).Methods("GET")
	r.HandleFunc("/todos", addTodo).Methods("POST")
	r.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE")

	// Serve static files from the "static" directory
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090" // Changed port to 9090 as per your logs
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}
