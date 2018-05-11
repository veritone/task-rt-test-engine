package main

import (
	// Built-in packages
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	// Local packages
	"github.com/veritone/task-rt-test-engine/models"

	// Veritone packages
	"github.com/veritone/edge-messages"
	vLogger "github.com/veritone/go-logger" // Veritone Go logger
	"github.com/veritone/go-messaging-lib/kafka"

	// External packages
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli" // CLI library
)

const (
	SigIntExitCode  = 130
	SigTermExitCode = 143

	ServiceName = "task-rt-test-engine"
)

type MsgCount struct {
	Total     int64 // total number of messages received (should equal successes + errors + ignored)
	Successes int64 // number of messages processed successfully
	Errors    int64 // number of messages processed with errors
	Ignored   int64 // number of messages not processed
}

func (mc *MsgCount) ToString() string {
	percentError := float32(0.0)
	if mc.Total != 0 {
		percentError = float32(mc.Errors) / float32(mc.Total)
	}
	return fmt.Sprintf("Message count: Total = %v, Successes = %v, Errors = %v, Ignored = %v, PercentError = %.2f%%",
		mc.Total, mc.Successes, mc.Errors, mc.Ignored, percentError*100)
}

var (
	// Values set at build time (see Makefile)
	BuildCommitHash string
	BuildTime       string

	ctx      models.AppContext
	msgCount MsgCount
)

func main() {
	ctx.App = cli.NewApp()
	ctx.App.Name = ServiceName
	ctx.App.Usage = "Lightweight engine to run with realtime heartbeat script"
	ctx.App.Version = "0.0.1 (" + runtime.Version() + ")"
	ctx.App.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, conf, c",
			Value:  "config/dev.json",
			Usage:  "Path to configuration file",
			EnvVar: "CONFIG_PATH",
		},
	}
	ctx.App.Action = func(c *cli.Context) {
		// Read configuration file
		err := loadConfig(&ctx.Config, c.GlobalString("config"))
		if err != nil {
			log.Fatal("Error reading configuration: " + err.Error())
		}

		// Initialize logger
		ctx.Logger, err = vLogger.New(os.Stdout,
			vLogger.GetLogLevel(ctx.Config.LogLevel), ctx.Config.LogFormat)
		if err != nil {
			log.Fatal("Error initializing logger: " + err.Error())
		}
		logrus.RegisterExitHandler(logrusExitHandler)
		ctx.Logger.Debug(fmt.Sprintf("Loaded service config settings:\n\t%+v\n\n", ctx.Config))

		brokers := strings.Split(ctx.Config.Kafka.Brokers, ",")
		ctx.Consumer, err = kafka.Consumer(ctx.Config.Kafka.ConsumerTopic, ctx.Config.Kafka.ConsumerGroupId, brokers...)
		if err != nil {
			log.Fatal("Error initializing kafka consumer: " + err.Error())
		}
		ctx.Producer, err = kafka.Producer(ctx.Config.Kafka.ProducerTopic, kafka.StrategyRoundRobin, brokers...)
		if err != nil {
			log.Fatal("Error initializing kafka producer: " + err.Error())
		}

		// Listen for SIGINT & SIGTERM and shutdown gracefully
		go listenForSignals()

		listenForJob()
	}
	ctx.App.Run(os.Args)
}

func listenForJob() {
	queue, err := ctx.Consumer.Consume(context.TODO(), kafka.ConsumerGroupOption)
	if err != nil {
		log.Fatal("Error initializing consumer: " + err.Error())
	}
	timer := time.NewTimer(ctx.Config.TTLinSec)

	for {
		select {
		case item := <-queue:
			msgCount.Total++

			// log the full message for debugging purpose
			ctx.Logger.Debug(string(item.Payload()))

			taskID, err := messages.GetTaskID(item)
			if err != nil || taskID == "" {
				errMsg := fmt.Sprintf("Received message without taskID: %v", string(item.Payload()))
				ctx.Logger.Error(errMsg)
				setChunkStatus("", "", messages.ChunkStatusError, errMsg, "")
				continue
			}

			msgType, err := messages.GetMsgType(item)
			if err != nil {
				errMsg := fmt.Sprintf("Received unknown message: %v", string(item.Payload()))
				ctx.Logger.Error(errMsg)
				setChunkStatus(taskID, "", messages.ChunkStatusError, errMsg, "")
				continue
			}

			if msgType != messages.MediaChunkType {
				errMsg := fmt.Sprintf("Not a media_chunk: %v", string(item.Payload()))
				ctx.Logger.Error(errMsg)
				setChunkStatus(taskID, "", messages.ChunkStatusError, errMsg, "")
				continue
			}

			var chunk messages.MediaChunk
			if err := json.Unmarshal(item.Payload(), &chunk); err != nil {
				errMsg := fmt.Sprintf("Unable to unmarshal event: %v", err)
				ctx.Logger.Error(errMsg)
				setChunkStatus(taskID, "", messages.ChunkStatusError, errMsg, "")
				continue
			}

			if chunk.ChunkUUID == "" {
				errMsg := fmt.Sprintf("Received message without ChunkUUID: %v", string(item.Payload()))
				ctx.Logger.Error(errMsg)
				setChunkStatus(taskID, "", messages.ChunkStatusError, errMsg, "")
				continue
			}

			if chunk.MimeType != "image/png" && chunk.MimeType != "image/jpeg" {
				errMsg := fmt.Sprintf("Not an image/png or image/jpeg: %v", chunk.MimeType)
				ctx.Logger.Warn(errMsg)
				setChunkStatus(taskID, chunk.ChunkUUID, messages.ChunkStatusIgnored, "", errMsg)
				continue
			}

			go generateEngineOutput(chunk)

			// Reset TTL
			timer.Reset(ctx.Config.TTLinSec)
		case <-timer.C:
			return
		default:
			if ctx.ShuttingDown {
				return
			}
		}
	}
}

func generateEngineOutput(chunk messages.MediaChunk) {
	var series []models.SeriesObject

	//Create a mock face series object for the aggregator
	face := models.SeriesObject{
		Start: chunk.StartOffsetMs,
		End:   chunk.EndOffsetMs,
		Object: models.Object{
			ObjectType: "face",
			Confidence: 1,
		},
	}
	series = append(series, face)

	seriesMap := map[string]interface{}{"series": series}
	seriesMapJson, err := json.Marshal(&seriesMap)
	if err != nil {
		setChunkStatus(chunk.TaskID, chunk.ChunkUUID, messages.ChunkStatusError, err.Error(), "")
		return
	}

	result := messages.EngineOutput{
		Type:          messages.EngineOutputType,
		TimestampUTC:  time.Now().UnixNano() / int64(time.Millisecond),
		TaskID:        chunk.TaskID,
		TDOID:         chunk.TDOID,
		JobID:         chunk.JobID,
		MIMEType:      chunk.MimeType,
		StartOffsetMs: chunk.StartOffsetMs,
		EndOffsetMs:   chunk.EndOffsetMs,
		Content:       string(seriesMapJson),
		ChunkUUID:     chunk.ChunkUUID,
	}

	resultJSON, err := json.Marshal(&result)
	if err != nil {
		setChunkStatus(chunk.TaskID, chunk.ChunkUUID, messages.ChunkStatusError, err.Error(), "")
		return
	}

	msg, err := kafka.NewMessage(chunk.TaskID, resultJSON)
	if err != nil {
		setChunkStatus(chunk.TaskID, chunk.ChunkUUID, messages.ChunkStatusError, err.Error(), "")
		return
	}

	err = ctx.Producer.Produce(context.Background(), msg)
	if err != nil {
		setChunkStatus(chunk.TaskID, chunk.ChunkUUID, messages.ChunkStatusError, err.Error(), "")
		return
	}
	fmt.Printf("Completed processing chunk: %s task: %s\n", chunk.JobID, chunk.TaskID)
}

// listenForSignals waits for SIGINT or SIGTERM to be captured.
// When caught, it shuts down gracefully and exits with the proper code.
func listenForSignals() {
	// Block until signal is caught
	notifyChan := make(chan os.Signal, 2)
	signal.Notify(notifyChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-notifyChan
	exitCode := 1
	switch sig {
	case syscall.SIGINT:
		exitCode = SigIntExitCode
	case syscall.SIGTERM:
		exitCode = SigTermExitCode
	}
	// Emit shutdown event, shutdown gracefully, exit with proper code
	gracefulShutdown()
	os.Exit(exitCode)
}

// logrusExitHandler emits a shutdown event, and shuts down gracefully.
func logrusExitHandler() {
	// logrus will call os.Exit(1)
	// https://github.com/sirupsen/logrus/blob/master/README.md#fatal-handlers
	ctx.Logger.Debug("Fatal level message logged!")
	gracefulShutdown()
}

// gracefulShutdown cleans up anything worth cleaning before exiting.
func gracefulShutdown() {
	ctx.Logger.Debug("Shutting down gracefully")
	ctx.ShuttingDown = true
}

func loadConfig(c *models.Config, cf string) error {
	// read config file
	var raw []byte
	var err error

	//raw, err = ioutil.ReadFile("/Users/home/go/src/github.com/veritone/task-rt-test-engine/config/dev.json")
	raw, err = ioutil.ReadFile(cf)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, &c)
	if err != nil {
		return err
	}

	engineID := os.Getenv("ENGINE_ID")
	if engineID != "" {
		c.EngineID = engineID
	}

	engineInstanceID := os.Getenv("ENGINE_INSTANCE_ID")
	if engineInstanceID != "" {
		c.EngineInstanceID = engineInstanceID
	}

	producerTopic := os.Getenv("KAFKA_CHUNK_TOPIC")
	if producerTopic != "" {
		c.Kafka.ProducerTopic = producerTopic
	}

	consumerTopic := os.Getenv("KAFKA_INPUT_TOPIC")
	if consumerTopic != "" {
		c.Kafka.ConsumerTopic = consumerTopic
	}

	consumerGroup := os.Getenv("KAFKA_CONSUMER_GROUP")
	if consumerGroup != "" {
		c.Kafka.ConsumerGroupId = consumerGroup
	}

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers != "" {
		c.Kafka.Brokers = brokers
	}

	baseUri := os.Getenv("VERITONE_API_BASE_URL")
	if baseUri != "" {
		c.VeritoneBaseUri = baseUri
	}

	ttlInSec := os.Getenv("END_IF_IDLE_SECS")
	if ttlInSec != "" {
		ttl, err := time.ParseDuration(ttlInSec + "s")
		if err != nil {
			return err
		}
		c.TTLinSec = ttl
	} else {
		// 30 min ttl
		c.TTLinSec = time.Minute * 30
	}

	return nil
}

func setChunkStatus(taskID string, chunkUUID string, status messages.ChunkStatus, errMsg string, infoMsg string) {
	chunkStatus := messages.EmptyChunkProcessedStatus()
	chunkStatus.TaskID = taskID
	chunkStatus.ChunkUUID = chunkUUID
	chunkStatus.Status = status
	chunkStatus.ErrorMsg = errMsg
	chunkStatus.InfoMsg = infoMsg

	ctx.Logger.Debugf("chunkStatus: %+v", chunkStatus)

	// print message count statistics
	switch status {
	case messages.ChunkStatusSuccess:
		msgCount.Successes++
	case messages.ChunkStatusError:
		msgCount.Errors++
	case messages.ChunkStatusIgnored:
		msgCount.Ignored++
	default:
		ctx.Logger.Errorf("Invalid chunk status: %v", status)
	}
	ctx.Logger.Info(msgCount.ToString())

	msg, err := chunkStatus.ToKafka()
	if err != nil {
		ctx.Logger.Errorf("failed to create Kafka message: %v", err)
		return
	}

	err = ctx.Producer.Produce(context.Background(), msg)
	if err != nil {
		ctx.Logger.Errorf("failed to produce Kafka message: %v", err)
		return
	}
}
