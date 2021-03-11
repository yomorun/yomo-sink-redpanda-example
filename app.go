package main

import (
	"context"
	"encoding/json"
	"fmt"

	kafka "github.com/segmentio/kafka-go"
	y3 "github.com/yomorun/y3-codec-golang"
	"github.com/yomorun/yomo/pkg/rx"
)

// NoiseDataKey represents the Tag of a Y3 encoded data packet
const NoiseDataKey = 0x10
const topic = "yomo-test"
const partition = 0

var (
	Conn *kafka.Conn = nil
)

func init() {
	var err error = nil
	// TODO os.env
	Conn, err = kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, partition)

	if err != nil {
		panic(err)
	}
}

type NoiseData struct {
	Noise float32 `y3:"0x11"`
	Time  int64   `y3:"0x12"`
	From  string  `y3:"0x13"`
}

// write to Redpanda
var produce = func(_ context.Context, i interface{}) (interface{}, error) {
	_, err := Conn.WriteMessages(
		kafka.Message{Value: i.([]byte)},
	)

	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("⚡️ %d successfully write to the redpanda", len(i.([]byte))), nil
}

// y3 callback
var callback = func(v []byte) (interface{}, error) {
	var mold NoiseData
	err := y3.ToObject(v, &mold)

	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(mold)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Handler will handle data in Rx way
func Handler(rxstream rx.RxStream) rx.RxStream {
	if Conn == nil {
		panic("not found kafka Conn.")
	}

	stream := rxstream.
		Subscribe(NoiseDataKey).
		OnObserve(callback).
		Map(produce).
		StdOut().
		Encode(NoiseDataKey)
	return stream
}
