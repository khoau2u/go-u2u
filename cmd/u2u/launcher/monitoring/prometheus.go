package monitoring

import (
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/unicornultrafoundation/go-u2u/monitoring/prometheus"
)

func SetupPrometheus(ctx *cli.Context) {
	if !metrics.Enabled {
		return
	}
	prometheus.SetNamespace("u2u")
	var endpoint = ctx.GlobalString(utils.MetricsPrometheusEndpointFlag.Name)
	prometheus.PrometheusListener(endpoint, nil)
}

var (
	// TODO: refactor it
	dbDirMonitor        atomic.Value
	dbSizeMetricMonitor = metrics.NewRegisteredFunctionalGauge("db_size", nil, measureDbDirMonitor)
)

func SetDataDirMonitor(datadir string) {
	dbDirMonitor.Store(datadir)
}

func measureDbDirMonitor() (size int64) {
	datadir, ok := dbDirMonitor.Load().(string)
	if !ok || datadir == "" || datadir == "inmemory" {
		return
	}

	err := filepath.Walk(datadir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	if err != nil {
		log.Error("filepath.Walk", "path", datadir, "err", err)
		return 0
	}

	return
}