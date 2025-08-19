package services

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
)

func UserExist(ctx context.Context, firestoreClient *firestore.Client, email string) (bool, error) {
	usersCollection := firestoreClient.Collection("Users")
	query := usersCollection.Where("email", "==", email).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return false, err // เกิด error ระหว่าง query
	}

	if len(docs) > 0 {
		return true, nil // email มีอยู่
	}

	return false, nil // email ไม่มี
}

func GetUserExist(ctx context.Context, firestoreClient *firestore.Client, email string) (*firestore.DocumentRef, error) {
	usersCollection := firestoreClient.Collection("Users")

	query := usersCollection.Where("email", "==", email).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err // เกิด error ระหว่าง query
	}

	if len(docs) == 0 {
		return nil, errors.New("user not found") // email ไม่มีในระบบ
	}

	// คืน docRef ของ document ที่เจอ
	return docs[0].Ref, nil

}
func GetUserData(ctx context.Context, firestoreClient *firestore.Client, email string) (*firestore.DocumentSnapshot, error) {
	usersCollection := firestoreClient.Collection("Users")

	// Query หา user ตาม email
	query := usersCollection.Where("email", "==", email).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err // เกิด error ระหว่าง query
	}

	if len(docs) == 0 {
		return nil, errors.New("user not found") // email ไม่มีในระบบ
	}

	// คืน DocumentSnapshot ของ document ที่เจอ
	return docs[0], nil
}
