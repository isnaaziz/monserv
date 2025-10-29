package notifier

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// KafkaPayload represents the JSON to be produced to Kafka
// Example fields follow the user's requested schema. Only Tag and Time are dynamic by default.
// Other fields can be configured via env vars KAFKA_* (see NewKafkaFromEnv).
// Numbers are kept as int where appropriate.

type KafkaPayload struct {
	IpAddress     string  `json:"ip_address"`
	Protocol      string  `json:"protocol"`
	IDRTU         string  `json:"id_rtu"`
	Action        string  `json:"action"`
	Address       string  `json:"address"`
	Channel       int     `json:"channel"`
	Type          string  `json:"type"`
	CA            int     `json:"ca"`
	Tag           string  `json:"tag"`   // diisi dari subject (UPPERCASE, tanpa spasi)
	Jenis         string  `json:"jenis"` // mapping jenis dari subject (MEMORY/DISK/CPU/…)
	Group         string  `json:"group"`
	Value         float64 `json:"value"`
	Time          int64   `json:"time"`
	GeneralAction int     `json:"general_action"`
	Subject       string  `json:"subject"` // subject asli ikut dikirim
}

// Kafka notifier using segmentio/kafka-go
// It reuses a single writer.

type Kafka struct {
	writer         *kafka.Writer
	defaultPayload KafkaPayload
	topic          string
}

func (k *Kafka) Name() string { return "kafka" }

func (k *Kafka) Send(subject, body string) error {
	if k == nil || k.writer == nil {
		return fmt.Errorf("kafka not configured")
	}
	p := k.defaultPayload
	p.Subject = subject
	p.Tag = sanitizeTag(subject)           // UPPERCASE + hapus spasi
	p.Jenis = mapJenisFromSubject(subject) // mapping jenis dari subject
	p.Time = time.Now().UnixMilli()

	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return k.writer.WriteMessages(ctx, kafka.Message{Value: b})
}

func NewKafkaFromEnv() (*Kafka, error) {
	brokers := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if brokers == "" {
		return nil, nil
	}
	var bs []string
	for _, s := range strings.Split(brokers, ",") {
		if v := strings.TrimSpace(s); v != "" {
			bs = append(bs, v)
		}
	}
	if len(bs) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS is empty")
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "datapoint"
	}

	username := os.Getenv("KAFKA_USERNAME")
	password := os.Getenv("KAFKA_PASSWORD")
	var mech sasl.Mechanism
	if username != "" || password != "" {
		mech = plain.Mechanism{Username: username, Password: password}
	}

	// TLS optional
	var tlsCfg *tls.Config
	if isTrue(os.Getenv("KAFKA_TLS_ENABLE")) {
		tlsCfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	clientID := os.Getenv("KAFKA_CLIENT_ID")

	writer := &kafka.Writer{
		Addr:         kafka.TCP(bs...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Transport: &kafka.Transport{
			SASL:     mech,
			TLS:      tlsCfg,
			ClientID: clientID,
		},
	}

	def := KafkaPayload{
		IpAddress:     getenvDefault("KAFKA_IP_ADDRESS", ""),
		Protocol:      getenvDefault("KAFKA_PROTOCOL", "TCP"),
		IDRTU:         getenvDefault("KAFKA_ID_RTU", "EXT-30890"),
		Action:        getenvDefault("KAFKA_ACTION", "SPONTANEUS"),
		Address:       getenvDefault("KAFKA_ADDRESS", "244444"),
		Channel:       atoiDefault(os.Getenv("KAFKA_CHANNEL"), 623),
		Type:          getenvDefault("KAFKA_TYPE", "GROUP10_VAR2"),
		CA:            atoiDefault(os.Getenv("KAFKA_CA"), 1),
		Tag:           "",
		Jenis:         "",
		Group:         getenvDefault("KAFKA_GROUP", "fe547e84-3b20-4811-ad51-3fb72d806159"),
		Value:         atofDefault(os.Getenv("KAFKA_VALUE"), 1),
		Time:          0,
		GeneralAction: atoiDefault(os.Getenv("KAFKA_GENERAL_ACTION"), 1),
		Subject:       "",
	}

	return &Kafka{writer: writer, defaultPayload: def, topic: topic}, nil
}

func sanitizeTag(s string) string {
	// UPPERCASE dan hapus semua spasi
	return strings.ToUpper(strings.ReplaceAll(s, " ", ""))
}

func mapJenisFromSubject(subject string) string {
	s := strings.ToUpper(subject)
	switch {
	case strings.Contains(s, "MEMORY BY") || strings.Contains(s, "PROC") || strings.Contains(s, "PROCESS"):
		return getenvDefault("KAFKA_JENIS_MEM_BY_APP", "MEMORY_BY_APLIKASI")
	case strings.Contains(s, "MEMORY"):
		return getenvDefault("KAFKA_JENIS_MEM_SERVER", "MEMORY_SERVER")
	case strings.Contains(s, "DISK"):
		return getenvDefault("KAFKA_JENIS_DISK_USAGE", "DISK_USAGE")
	case strings.Contains(s, "CPU"):
		return getenvDefault("KAFKA_JENIS_CPU_USAGE", "CPU_USAGE")
	default:
		return getenvDefault("KAFKA_JENIS_DEFAULT", "ALERT_GENERIC")
	}
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoiDefault(v string, def int) int {
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func atofDefault(v string, def float64) float64 {
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
