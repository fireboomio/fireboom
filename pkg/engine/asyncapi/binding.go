package asyncapi

type BindingKey string

const (
	BindingKey_http         BindingKey = "http"
	BindingKey_ws           BindingKey = "ws"
	BindingKey_kafka        BindingKey = "kafka"
	BindingKey_anypointmq   BindingKey = "anypointmq"
	BindingKey_amqp         BindingKey = "amqp"
	BindingKey_mqtt         BindingKey = "mqtt"
	BindingKey_mqtt5        BindingKey = "mqtt5"
	BindingKey_nats         BindingKey = "nats"
	BindingKey_jms          BindingKey = "jms"
	BindingKey_sns          BindingKey = "sns"
	BindingKey_solace       BindingKey = "solace"
	BindingKey_sqs          BindingKey = "sqs"
	BindingKey_stomp        BindingKey = "stomp"
	BindingKey_redis        BindingKey = "redis"
	BindingKey_mercure      BindingKey = "mercure"
	BindingKey_ibmmq        BindingKey = "ibmmq"
	BindingKey_googlepubsub BindingKey = "googlepubsub"
	BindingKey_pulsar       BindingKey = "pulsar"
)
