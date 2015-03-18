/*
Package ghealth provides a Gin middleware
to gocraft/health performance monitoring toolkit.

By default it creates StatsD sink, falling back to stdout if
error happened or StatsD server was not provided.

Recovery is supported and panics are sent as general errors
with request's URI.

Example

	func main() {
		// Standard router initialization
		router := gin.Default()

		// First, you need to create a new stream...

		// Simplest sink, stdout only
		hstream := ghealth.NewStream("", "", "")

		// STDOUT and JSON sinks; creates independent http server on port 5020
		hstream := ghealth.NewStream("", "", "127.0.0.1:5020")

		// StatsD and JSON sinks
		hstream := ghealth.NewStream("statsd.server:5000", "yourappname", "127.0.0.1:5020")

		// It's a standard *health.Stream, so you can do anything you want!
		hstream.AddSink(&health.WriterSink{os.Stdout})


		router.Use(ghealth.Health(hstream))
		...
	}

	var someRoute gin.HandlerFunc = func(c *gin.Context) {
		// Retrieve a job object
		job := ghealth.Job(c, "some_route")

		// It's a *health.Job, read health godoc for more info :)
		job.Event("some_event")
	}

At the moment it wasn't heavily tested. I'm still not sure how health
implements concurrency, but as long as stream is thread-safe - ghealth
should be, too.
*/
package ghealth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gocraft/health"
	"net/http"
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

// Use this method to inject the middleware and recovery.
func Health(stream *health.Stream) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rval := recover(); rval != nil {
				stream.EventErr(fmt.Sprintf("Panic at %v", c.Request.RequestURI), rval.(error))
				c.Writer.WriteHeader(http.StatusInternalServerError)
			}
		}()
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
