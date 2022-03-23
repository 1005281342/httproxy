package main

import (
	"flag"
	"net/http"

	"github.com/1005281342/httproxy"
	"github.com/polarismesh/polaris-go/api"
)

var (
	namespace string
	gConsumer api.ConsumerAPI
)

func initArgs() {
	flag.StringVar(&namespace, "namespace", "default", "namespace")
}

func main() {
	initArgs()
	flag.Parse()

	var consumer, err = api.NewConsumerAPI()
	if err != nil {
		panic(err)
	}
	gConsumer = consumer

	if err := http.ListenAndServe("127.0.0.1:2338", httproxy.New(namespace, gConsumer,
		httproxy.WithProm("dev", "./boot.yaml"))); err != nil {
		panic(err)
	}
}
