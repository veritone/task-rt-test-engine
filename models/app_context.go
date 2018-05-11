package models

import (
	// Veritone packages
	vLogger "github.com/veritone/go-logger" // Veritone Go logger

	"github.com/prometheus/client_golang/prometheus" // Prometheus client library
	"github.com/urfave/cli"
	"github.com/veritone/go-messaging-lib"
)

type AppContext struct {
	App                    *cli.App
	Config                 Config
	Logger                 *vLogger.Logger
	BuildInfoGaugeVec      *prometheus.GaugeVec
	TranslationJobsCounter prometheus.Counter
	Producer               messaging.Producer
	Consumer               messaging.Consumer
	ShuttingDown           bool
}
