package ghealth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gocraft/health"
	"os"
	"time"
)

const (
	defaultStreamKey = "github.com/utrack/gin-health"
	defaultJobKey    = "github.com/utrack/gin-health|job"
)

// NewStream initializes health sink to statsd using supplied
// statsd address (IP:PORT) and appname.
// Falls back to stdout if none supplied.
// Also creates Json sink for healthd at supplied address
// (serversink) if not empty.
//
// statsd: StatsD address and port.
//
// appname Application name for StatsD.
//
// serversink: Bind address for Json sink, empty if not needed.
func NewStream(statsd string, appname string, serversink string) *health.Stream {
	var stream = health.NewStream()

	if statsd != "" {
		statsdSink, err := health.NewStatsDSink(statsd, appname)
		if err != nil {
			fmt.Println("HEALTH: Adding stdout health sink...")
			stream.AddSink(&health.WriterSink{os.Stdout})
			stream.EventErr("new_statsd_sink", err)
		} else {
			fmt.Println("HEALTH: Adding statsd health sink...")
			stream.AddSink(statsdSink)
		}
	} else {
		fmt.Println("HEALTH: Adding stdout health sink...")
		stream.AddSink(&health.WriterSink{os.Stdout})
	}

	if serversink != "" {
		sink := health.NewJsonPollingSink(time.Minute, time.Minute*5)
		stream.AddSink(sink)
		sink.StartServer(serversink)
		fmt.Println("HEALTH: Adding json health sink...")
	}
	return stream
}

// Use this method to inject the middleware.
func Health(stream *health.Stream) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(defaultStreamKey, stream)
		c.Next()
	}
}

// Job creates a new job with given name.
//
// c: current Gin context.
//
// name: Job's name.
func Job(c *gin.Context, name string) *health.Job {
	job := c.MustGet(defaultStreamKey).(*health.Stream).NewJob(name)
	c.Set(defaultJobKey, time.Now())
	return job
}

// TimeSince is a little helper over time.Time.TimeSince
// which returns the interval in nanoseconds,
// suitable for feeding to job.Timing.
func TimeSince(t time.Time) int64 {
	return time.Since(t).Nanoseconds()
}
