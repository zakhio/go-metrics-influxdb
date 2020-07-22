go-metrics-influxdb
===================

This is a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library which will post the metrics to [InfluxDB](https://influxdb.com/).

Based on the official [InfluxDB Client Go](https://github.com/influxdata/influxdb-client-go) and compatible with InfluxDB 2.x and InfluxDB 1.8+.

Installation
------------

```shell
go get https://github.com/zakhio/go-metrics-influxdb
```

Usage
-----

```go
import "github.com/zakhio/go-metrics-influxdb"

go reporter.InfluxDBWithTags(
    metrics.DefaultRegistry,    // metrics registry
    time.Second * 10,           // reporting interval
    serverURL,                  // InfluxDB instance url
    organizationID,             // organization id
    bucketID,                   // data bucket id
    measurement,                // measurement
    token,                      // access token
    tags,                       // default tags (for example, server name, artifact version, etc)
    alignTimestamps             // flag to align the timestamps
)
```

Metrics can be aligned to the beginning of a bucket as defined by the interval.

Setting `alignTimestamps` to `true` will cause the timestamp to be truncated down to the nearest even integral of the reporting interval.

For example, if the interval is 30 seconds, timestamps will be aligned on `:00` and `:30` for every reporting interval.

Note: check [go-metrics-influxdb-grpc-example](https://github.com/zakhio/go-metrics-influxdb-grpc-example) for more hands on example. 

License
-------

go-metrics-influxdb is licensed under the MIT license. See the LICENSE file for details.
