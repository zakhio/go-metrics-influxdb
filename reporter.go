package reporter

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	client "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api/write"
	"github.com/rcrowley/go-metrics"
)

type reporter struct {
	registry       metrics.Registry
	interval       time.Duration
	align          bool
	serverURL      string
	organizationId string
	bucketId       string

	measurement string
	token       string
	tags        map[string]string

	client client.Client
}

// InfluxDB starts a InfluxDB reporter which will post the metrics
// from the given registry at each interval.
func InfluxDB(registry metrics.Registry, interval time.Duration, serverUrl, organizationID, bucketID, measurement, token string, alignTimestamps bool) {
	InfluxDBWithTags(registry, interval, serverUrl, organizationID, bucketID, measurement, token, map[string]string{}, alignTimestamps)
}

// InfluxDBWithTags starts a InfluxDB reporter which will post the metrics
// from the given registry at each 'd' interval with the specified 'tags'.
func InfluxDBWithTags(registry metrics.Registry, interval time.Duration, serverUrl, organizationID, bucketID, measurement, token string, tags map[string]string, alignTimestamps bool) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		log.Printf("unable to parse InfluxDB serverURL %s. err=%v", serverUrl, err)
		return
	}

	rep := &reporter{
		registry:       registry,
		interval:       interval,
		serverURL:      u.String(),
		organizationId: organizationID,
		bucketId:       bucketID,
		measurement:    measurement,
		token:          token,
		tags:           tags,
		align:          alignTimestamps,
	}
	rep.makeClient()
	rep.run()
}

func (r *reporter) makeClient() {
	r.client = client.NewClient(r.serverURL, r.token)
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				log.Printf("unable to send metrics to InfluxDB. err=%v", err)
			}
		case <-pingTicker:
			ready, err := r.client.Ready(context.Background())

			if !ready {
				log.Printf("got error while sending a ping to InfluxDB, trying to recreate client. err=%v", err)
				r.makeClient()
			}
		}
	}
}

func (r *reporter) send() error {
	var pts []*write.Point

	now := time.Now()
	if r.align {
		now = now.Truncate(r.interval)
	}

	r.registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			pts = append(pts, write.NewPoint(
				r.measurement,
				r.tags,
				map[string]interface{}{
					fmt.Sprintf("%s.count", name): ms.Count(),
				},
				now,
			))
		case metrics.Gauge:
			ms := metric.Snapshot()
			pts = append(pts, write.NewPoint(
				r.measurement,
				r.tags,
				map[string]interface{}{
					fmt.Sprintf("%s.gauge", name): ms.Value(),
				},
				now,
			))
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			pts = append(pts, write.NewPoint(
				r.measurement,
				r.tags,
				map[string]interface{}{
					fmt.Sprintf("%s.gauge", name): ms.Value(),
				},
				now,
			))
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
			fields := map[string]float64{
				"count":    float64(ms.Count()),
				"max":      float64(ms.Max()),
				"mean":     ms.Mean(),
				"min":      float64(ms.Min()),
				"stddev":   ms.StdDev(),
				"variance": ms.Variance(),
				"p50":      ps[0],
				"p75":      ps[1],
				"p95":      ps[2],
				"p99":      ps[3],
			}
			for k, v := range fields {
				pts = append(pts, write.NewPoint(
					r.measurement,
					bucketTags(k, r.tags),
					map[string]interface{}{
						fmt.Sprintf("%s.histogram", name): v,
					},
					now,
				))

			}
		case metrics.Meter:
			ms := metric.Snapshot()
			fields := map[string]float64{
				"count": float64(ms.Count()),
				"m1":    ms.Rate1(),
				"m5":    ms.Rate5(),
				"m15":   ms.Rate15(),
				"mean":  ms.RateMean(),
			}
			for k, v := range fields {
				pts = append(pts, write.NewPoint(
					r.measurement,
					bucketTags(k, r.tags),
					map[string]interface{}{
						fmt.Sprintf("%s.meter", name): v,
					},
					now,
				))
			}

		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
			fields := map[string]float64{
				"count":    float64(ms.Count()),
				"max":      float64(ms.Max()),
				"mean":     ms.Mean(),
				"min":      float64(ms.Min()),
				"stddev":   ms.StdDev(),
				"variance": ms.Variance(),
				"p50":      ps[0],
				"p75":      ps[1],
				"p95":      ps[2],
				"p99":      ps[3],
				"m1":       ms.Rate1(),
				"m5":       ms.Rate5(),
				"m15":      ms.Rate15(),
				"meanrate": ms.RateMean(),
			}
			for k, v := range fields {
				pts = append(pts, write.NewPoint(
					r.measurement,
					bucketTags(k, r.tags),
					map[string]interface{}{
						fmt.Sprintf("%s.timer", name): v,
					},
					now,
				))
			}
		}
	})

	err := r.client.WriteAPIBlocking(r.organizationId, r.bucketId).WritePoint(context.Background(), pts...)
	return err
}

func bucketTags(bucket string, tags map[string]string) map[string]string {
	m := map[string]string{}
	for tk, tv := range tags {
		m[tk] = tv
	}
	m["bucketId"] = bucket
	return m
}
