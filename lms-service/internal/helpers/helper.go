package helpers

import (
	"fmt"
	"github.com/rs/zerolog"
	_ "github.com/rs/zerolog"
	"github.com/streadway/amqp"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func BadRequestResponse(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func ServerErrorResponse(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func NotFoundResponse(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
}

func WriteJSON(c *gin.Context, statusCode int, data gin.H) {
	c.JSON(statusCode, data)
}

func ReadIDParam(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func PublishMessage(ch *amqp.Channel, queueName string, message string) error {
	err := ch.Publish(
		"",        // Empty exchange name for routing to the declared queue
		queueName, // Target queue name
		false,     // Mandatory flag (optional, can be set to true for guaranteed delivery)
		false,     // Immediate flag (optional, can be set to true to skip the publisher confirmation)
		amqp.Publishing{
			ContentType: "text/json",
			Body:        []byte(message),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func ConnectToRabbitMQ() (*amqp.Channel, error) {
	conn, err := amqp.Dial("amqp://localhost:5672/") // Adjust connection details if needed
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return ch, nil
}

func DeclareQueue(ch *amqp.Channel, queueName string) error {
	_, err := ch.QueueDeclare(
		queueName, // Name of the queue
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	return nil
}

func SendMessageToQueue(l zerolog.Logger, ch *amqp.Channel, message string) {
	err := DeclareQueue(ch, "notification_queue")
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to declare a queue")
	}
	defer ch.Close()

	err = PublishMessage(ch, "notification_queue", message)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to publish a message")
	}
}
