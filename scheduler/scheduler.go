package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"github.com/streadway/amqp"
	"github.com/twinj/uuid"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

//{
//  "broker": "",
//  "exchange": {
//    "name": "scaleworks",
//    "type": "topic",
//    "durable": "true"
//  },
//  "jobs": [
//  { "name": "sync_nodes",
//    "interval": "3m",
//    "routing_key": "sync_nodes"
//  },
//  { "name": "sync_clouds",
//    "interval": "1d",
//    "routing_key": "sync_clouds"
//  }
//  ]
//}

//Struct config
type Config struct {
	Broker   string
	Exchange ExchangeType
	Jobs     []JobsType
}

type ExchangeType struct {
	Name    string
	Type    string
	Durable string
}

type JobsType struct {
	Name        string
	Interval    string
	Routing_key string
}

//Faile Error
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func DistributeJob(job JobsType, exchange_name string, ch) {

	now := time.Now()

	tracking_id := uuid.NewV4()

	body := "{\"tracking_id\":\"" + tracking_id.String() + "\", \"job_name\":\"" + job.Name + "\", \"when_distributed\":\"" + fmt.Sprintf("%s", now) + "\"}"

	err = ch.Publish(
		exchange_name,   // exchange
		job.Routing_key, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	log.Println("Job " + body + " distributed.")

	failOnError(err, "Job "+body+" failed to distribute.")

}

func GetConfig() Config {
	flag.Parse()
	file := flag.Arg(0)

	filename, err := filepath.Abs(file)
	if err != nil {
		log.Fatalf("Can't find the file " + filename)
		os.Exit(1)
	}

	jsonFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	config := Config{}

	err = json.Unmarshal(jsonFile, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("Loading Config as " + string(jsonFile))

	return config
}

func main() {

	config := GetConfig()

	c := cron.New()
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	for _, job := range config.Jobs {
		realjob := job
		c.AddFunc("@every "+realjob.Interval, func() {
			DistributeJob(realjob, config.Exchange.Name, ch)
		})
		log.Println("Job " + job.Name + " has been scheduled on every " + job.Interval)
	}

	c.Start()
	select {}
}