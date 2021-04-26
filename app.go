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

	y3 "github.com/yomorun/y3-codec-golang"
	"github.com/yomorun/yomo/pkg/rx"
)

// noiseDataKey represents the Tag of a Y3 encoded data packet.
const noiseDataKey = 0x10

// batchSize is the amount of data that will be inserted into Redpanda in batch.
const batchSize = 100

// bufferMilliseconds is the time in milliseconds that the data will be buffered and inserted into Redpanda in batch.
const bufferMilliseconds = 3e3

var (
	pandaProxyURL = "" // Redpanda Proxy URL.
	topic         = "" // Topic name.
)

func init() {
	pandaProxyURL = os.Getenv("PANDAPROXY_URL")
	if pandaProxyURL == "" {
		pandaProxyURL = "http://localhost:8082"
	}
	topic = os.Getenv("REDPANDA_TOPIC")
	if topic == "" {
		topic = "yomo-test"
	}
}

// noiseData represents the structure of data.
type noiseData struct {
	Noise float32 `y3:"0x11" json:"noise"`
	Time  int64   `y3:"0x12" json:"time"`
	From  string  `y3:"0x13" json:"from"`
}

// postData represents the structure of records that will be posted to Redpanda.
type postData struct {
	Records []postDataItem `json:"records"`
}

// postDataItem represents the structure of data item.
type postDataItem struct {
	Value     interface{} `json:"value"`
	Partition int         `json:"partition"`
}

// getPostBody gets the body of HTTP POST to Redpanda Proxy.
func getPostBody(data []interface{}) ([]byte, error) {
	items := make([]postDataItem, len((data)))
	for i, noise := range data {
		items[i] = postDataItem{
			Value:     noise,
			Partition: 0,
		}
	}
	return json.Marshal(postData{
		Records: items,
	})
}

// write data to Redpanda Proxy in batch via RESTful API.
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
	resp, err := http.Post(fmt.Sprintf("%s/topics/%s", pandaProxyURL, topic), "application/vnd.kafka.binary.v2+json", bytes.NewBuffer(postBody))
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
	// print the response
	log.Printf(string(body))

	return fmt.Sprintf("⚡️ write %d items to redpanda successfully", len(data)), nil
}

// y3 callback
var callback = func(v []byte) (interface{}, error) {
	var mold noiseData
	// decode bytes by Y3 Codec.
	err := y3.ToObject(v, &mold)

	if err != nil {
		return nil, err
	}

	// return the JSON encoding for insertion in Redpanda.
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
		BufferWithTimeOrCount(bufferMilliseconds, batchSize).
		Map(produce).
		StdOut().
		Encode(noiseDataKey)
	return stream
}
