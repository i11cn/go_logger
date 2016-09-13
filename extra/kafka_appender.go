package extra

import (
	"fmt"
	"github.com/Shopify/sarama"
	"go_logger"
)

type (
	KafkaAppender struct {
		logger.BaseAppender
		producer sarama.AsyncProducer
		topic    string
		buf      chan *sarama.ProducerMessage
		sync     bool
	}
)

func (k *KafkaAppender) Write(msg []byte) {
	s := &sarama.ProducerMessage{Topic: k.topic, Value: sarama.ByteEncoder(msg)}
	if k.sync {
		k.producer.Input() <- s
	} else {
		select {
		case k.buf <- s:
		default:
			fmt.Println("Kafka系统忙，发送失败: ", string(msg))
		}
	}
}

func NewKafkaAppender(prod sarama.AsyncProducer, topic, layout string) *KafkaAppender {
	ret := &KafkaAppender{logger.NewBaseAppender(layout), prod, topic, make(chan *sarama.ProducerMessage, 100), true}
	if ret.sync {
		go func() {
			select {
			case msg := <-ret.buf:
				ret.producer.Input() <- msg
			case e := <-ret.producer.Errors():
				fmt.Println("Kafka消息发送失败：", e)
			}
		}()
	}
	return ret
}
