package mb_cud_publisher

import (
	"sync"

	"github.com/rabbitmq/amqp091-go"

	mb_cud_queue_provisioning "github.com/anan112pcmec/Burung-backend-1/app/message_broker/provisioning/cud_exchange/queue"
)

type Publisher struct {
	Ch          *amqp091.Channel
	QueueCreate *mb_cud_queue_provisioning.CreateQueue
	QueueUpdate *mb_cud_queue_provisioning.UpdateQueue
	QueueDelete *mb_cud_queue_provisioning.DeleteQueue
	Mu          sync.Mutex
}
