package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/reactivex/rxgo/v2"
	y3 "github.com/yomorun/y3-codec-golang"
	"github.com/yomorun/yomo/pkg/rx"
)

// noiseDataKey represents the Tag of a Y3 encoded data packet
const noiseDataKey = 0x10
const batchSize = 1000
const bufferSeconds = 30

var (
	pandaproxyURL = ""
	topic         = ""
	bufferTime    = rxgo.WithDuration(bufferSeconds * time.Second)
)

func init() {
	pandaproxyURL = os.Getenv("PANDAPROXY_URL")
	if pandaproxyURL == "" {
		pandaproxyURL = "http://localhost:8082"
	}
	topic = os.Getenv("REDPANDA_TOPIC")
	if topic == "" {
		topic = "yomo-test"
	}
}

type noiseData struct {
	Noise float32 `y3:"0x11" json:"noise"`
	Time  int64   `y3:"0x12" json:"time"`
	From  string  `y3:"0x13" json:"from"`
}

type postData struct {
	Records []noiseDataItem `json:"records"`
}

type noiseDataItem struct {
	Value     interface{} `json:"value"`
	Partition int         `json:"partition"`
}

// get POST body to Redpanda Proxy
func getPostBody(data []interface{}) ([]byte, error) {
	items := make([]noiseDataItem, len((data)))
	for i, noise := range data {
		items[i] = noiseDataItem{
			Value:     noise,
			Partition: 0,
		}
	}
	return json.Marshal(postData{
		Records: items,
	})
}

// write to Redpanda
var produce = func(_ context.Context, v interface{}) (interface{}, error) {
	data, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("v is not a slice")
	}
	postBody, err := getPostBody(data)
	if err != nil {
		return nil, err
	}

	// post data to Redpanda
	resp, err := http.Post(fmt.Sprintf("%s/topics/%s", pandaproxyURL, topic), "application/vnd.kafka.binary.v2+json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	defer resp.Body.Close()
	// read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	log.Printf(string(body))

	return fmt.Sprintf("⚡️ write %d items to redpanda successfully", len(data)), nil
}

// y3 callback
var callback = func(v []byte) (interface{}, error) {
	var mold noiseData
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
	stream := rxstream.
		Subscribe(noiseDataKey).
		OnObserve(callback).
		BufferWithTimeOrCount(bufferTime, batchSize).
		Map(produce).
		StdOut().
		Encode(noiseDataKey)
	return stream
}
