# yomo-sink-redpanda-example

Redpanda 🙌 YoMo. This exmpale demonstrates how to integrate Redpanda to YoMo and bulk produce messages into Redpanda after stream processing.

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

### Install YoMo CLI

Please visit [YoMo Getting Started](https://github.com/yomorun/yomo#1-install-cli).

## Quick Start

## Create a topic in Redpanda

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
```

### Run your serverless app in development

`PANDAPROXY_URL` is the URL of your `PandaProxy` in Redpanda.

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
