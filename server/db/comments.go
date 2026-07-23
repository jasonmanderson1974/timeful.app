package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sirtom/server/logger"
	"sirtom/server/models"
)

// GetComments returns an event's discussion thread, oldest first (C7).
func GetComments(eventId string) ([]models.Comment, error) {
	objectId, err := primitive.ObjectIDFromHex(eventId)
	if err != nil {
		// eventId is malformatted
		return []models.Comment{}, nil
	}

	result, err := CommentsCollection.Find(
		context.Background(),
		bson.M{"eventId": objectId},
		options.Find().SetSort(bson.M{"createdAt": 1}),
	)
	if err != nil {
		logger.StdErr.Println(err)
		return []models.Comment{}, err
	}

	var comments []models.Comment
	if err := result.All(context.Background(), &comments); err != nil {
		logger.StdErr.Println(err)
		return []models.Comment{}, err
	}

	return comments, nil
}

// GetCommentById returns a single comment, or nil if it doesn't exist.
func GetCommentById(commentId string) (*models.Comment, error) {
	objectId, err := primitive.ObjectIDFromHex(commentId)
	if err != nil {
		return nil, nil
	}
	result := CommentsCollection.FindOne(context.Background(), bson.M{"_id": objectId})
	var comment models.Comment
	if err := result.Decode(&comment); err != nil {
		return nil, nil // not found
	}
	return &comment, nil
}

func InsertComment(comment models.Comment) error {
	_, err := CommentsCollection.InsertOne(context.Background(), comment)
	if err != nil {
		logger.StdErr.Println(err)
	}
	return err
}

func UpdateCommentText(commentId primitive.ObjectID, text string, updatedAt primitive.DateTime) error {
	_, err := CommentsCollection.UpdateByID(
		context.Background(),
		commentId,
		bson.M{"$set": bson.M{"text": text, "updatedAt": updatedAt}},
	)
	if err != nil {
		logger.StdErr.Println(err)
	}
	return err
}

func DeleteComment(commentId primitive.ObjectID) error {
	_, err := CommentsCollection.DeleteOne(context.Background(), bson.M{"_id": commentId})
	if err != nil {
		logger.StdErr.Println(err)
	}
	return err
}
