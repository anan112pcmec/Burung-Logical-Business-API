package mb_cud_publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"

	mb_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer"
)

func CreatePublish[
	T mb_serializer.PublishPayloadJson | mb_serializer.PublishPayloadProto,
](
	ctx context.Context,
	Publish *Publisher,
	Payload T,
) error {

	var (
		body        []byte
		contentType string
		headers     amqp091.Table
	)

	switch v := any(Payload).(type) {

	case mb_serializer.PublishPayloadJson:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("json marshal failed: %w", err)
		}
		body = b
		contentType = "application/json"

	case mb_serializer.PublishPayloadProto:
		body = v.Payload
		contentType = "application/x-protobuf"
		headers = amqp091.Table{
			"table_name": v.TableName,
		}

	default:
		return fmt.Errorf("unsupported payload type")
	}

	// 🔒 Lock channel supaya thread-safe
	Publish.Mu.Lock()
	defer Publish.Mu.Unlock()

	return Publish.Ch.PublishWithContext(
		ctx,
		Publish.QueueCreate.ExchangeName,
		Publish.QueueCreate.QueueBind,
		true,
		false,
		amqp091.Publishing{
			ContentType: contentType,
			Body:        body,
			Headers:     headers,
		},
	)
}

func UpdatePublish[
	T mb_serializer.PublishPayloadJson | mb_serializer.PublishPayloadProto,
](
	ctx context.Context,
	Publish *Publisher,
	Payload T,
) error {

	var (
		body        []byte
		contentType string
		headers     amqp091.Table
	)

	switch v := any(Payload).(type) {

	case mb_serializer.PublishPayloadJson:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("json marshal failed: %w", err)
		}
		body = b
		contentType = "application/json"

	case mb_serializer.PublishPayloadProto:
		body = v.Payload
		contentType = "application/x-protobuf"
		headers = amqp091.Table{
			"table_name": v.TableName,
		}

	default:
		return fmt.Errorf("unsupported payload type")
	}

	// 🔒 Lock channel supaya thread-safe
	Publish.Mu.Lock()
	defer Publish.Mu.Unlock()

	return Publish.Ch.PublishWithContext(
		ctx,
		Publish.QueueUpdate.ExchangeName,
		Publish.QueueUpdate.QueueBind,
		true,
		false,
		amqp091.Publishing{
			ContentType: contentType,
			Body:        body,
			Headers:     headers,
		},
	)
}

func DeletePublish[
	T mb_serializer.PublishPayloadJson | mb_serializer.PublishPayloadProto,
](
	ctx context.Context,
	Publish *Publisher,
	Payload T,
) error {

	var (
		body        []byte
		contentType string
		headers     amqp091.Table
	)

	switch v := any(Payload).(type) {

	case mb_serializer.PublishPayloadJson:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("json marshal failed: %w", err)
		}
		body = b
		contentType = "application/json"

	case mb_serializer.PublishPayloadProto:
		body = v.Payload
		contentType = "application/x-protobuf"
		headers = amqp091.Table{
			"table_name": v.TableName,
		}

	default:
		return fmt.Errorf("unsupported payload type")
	}

	// 🔒 Lock channel supaya thread-safe
	Publish.Mu.Lock()
	defer Publish.Mu.Unlock()

	return Publish.Ch.PublishWithContext(
		ctx,
		Publish.QueueDelete.ExchangeName,
		Publish.QueueDelete.QueueBind,
		true,
		false,
		amqp091.Publishing{
			ContentType: contentType,
			Body:        body,
			Headers:     headers,
		},
	)
}
