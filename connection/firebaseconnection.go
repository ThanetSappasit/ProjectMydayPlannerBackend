package connection

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var FirestoreClient *firestore.Client

func InitFirestoreClient() (*firestore.Client, error) {
	// Load .env for local dev environment (skip if on production)
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found — assuming ENV vars are set in Render")
	}

	// Get service account path from ENV
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is not set in the environment")
	}

	// Initialize Firebase with Service Account credentials
	ctx := context.Background()
	sa := option.WithCredentialsFile(credsFile)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firestore client: %v", err)
	}

	log.Println("Firebase initialized successfully")
	return client, nil
}
