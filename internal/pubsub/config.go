package pubsub

type Config struct {
	Brokers []string         `json:"brokers"`
	Topics  map[string][]int `json:"topics"`
}
