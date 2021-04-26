# yomo-sink-redpanda-example

Redpanda 🙌 YoMo. This exmpale demonstrates how to integrate Redpanda to YoMo and bulk produce messages into Redpanda after stream processing by YoMo.

## About Redpanda

[Redpanda](https://github.com/vectorizedio/redpanda) is a streaming platform for mission critical workloads. Kafka® compatible, No Zookeeper®, no JVM, and no code changes required. Use all your favorite open source tooling - 10x faster.

Redpanda are building a real-time streaming engine for modern applications - from the enterprise to the solo dev prototyping a react application on her laptop. Redpanda go beyond the Kafka protocol, into the future of streaming with inline WASM transforms and geo-replicated hierarchical storage. A new platform that scales with you from the smallest projects to petabytes of data distributed across the globe.

For more information, please visit [Vectorized homepage](https://vectorized.io/).

## About YoMo

[YoMo](https://github.com/yomorun/yomo) is an open-source Streaming Serverless Framework for building Low-latency Edge Computing applications. Built atop QUIC Transport Protocol and Functional Reactive Programming interface. makes real-time data processing reliable, secure, and easy.

For more information, please visit [YoMo homepage](https://yomo.run/).

## Prerequisites

### Install Redpanda

Please visit [Redpanda Getting Started](https://vectorized.io/docs/quick-start-linux) and enable [PandaProxy](https://github.com/vectorizedio/redpanda/pull/682) after installation.

This example will use HTTP REST API in `PandaProxy` to produce the message.

### Install YoMo CLI

Please visit [YoMo Getting Started](https://github.com/yomorun/yomo#1-install-cli).

## Quick Start

### Create a topic in Redpanda

We’ll call it "yomo-test":

```bash
rpk topic create yomo-test
```

### Create your serverless app

Please visit [YoMo homepage](https://github.com/yomorun/yomo#2-create-your-serverless-app).

### Copy `app.go` to your serverless app

```go
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
```

### Run your serverless app in development

`PANDAPROXY_URL` is the URL of `PandaProxy` in Redpanda.

```bash
$ PANDAPROXY_URL=http://127.0.0.1:8082 yomo dev
2021/03/18 14:13:13 Building the Serverless Function File...
2021/03/18 14:13:16 ✅ Listening on 0.0.0.0:4242
2021/03/18 14:13:19 {"offsets":[{"partition":0,"offset":113}]}
[StdOut]:  ⚡️ write 10 items to redpanda successfully
2021/03/18 14:13:20 {"offsets":[{"partition":0,"offset":123}]}
[StdOut]:  ⚡️ write 10 items to redpanda successfully
2021/03/18 14:13:21 {"offsets":[{"partition":0,"offset":133}]}
[StdOut]:  ⚡️ write 10 items to redpanda successfully
2021/03/18 14:13:22 {"offsets":[{"partition":0,"offset":143}]}
[StdOut]:  ⚡️ write 10 items to redpanda successfully
```

### Consume the message in Redpanda

```bash
rpk topic consume yomo-test
```

You will see messages like this:

```json
{
 "message": "{\"noise\":153.04851,\"time\":1616048002511,\"from\":\"127.0.0.1\"}",
 "partition": 0,
 "offset": 150,
 "timestamp": "2021-03-18T06:13:22.888Z"
}
{
 "message": "{\"noise\":43.49945,\"time\":1616048002611,\"from\":\"127.0.0.1\"}",
 "partition": 0,
 "offset": 151,
 "timestamp": "2021-03-18T06:13:22.888Z"
}
{
 "message": "{\"noise\":85.69591,\"time\":1616048002711,\"from\":\"127.0.0.1\"}",
 "partition": 0,
 "offset": 152,
 "timestamp": "2021-03-18T06:13:22.888Z"
}
```
