package integration

import (
	"context"
	"io"
	"net/url"
	"time"

	kineendpoint "github.com/k3s-io/kine/pkg/endpoint"
	"sigs.k8s.io/controller-runtime/pkg/internal/testing/integration/internal"
)

// Kine knows how to run an etcd server.
type Kine struct {
	// URL is the address the Kine should listen on for client connections.
	//
	// If this is not specified, we default to a random free port on localhost.
	URL *url.URL

	// StartTimeout, StopTimeout specify the time the Kine is allowed to
	// take when starting and stopping before an error is emitted.
	//
	// If not specified, these default to 20 seconds.
	StartTimeout time.Duration
	StopTimeout  time.Duration

	// Out, Err specify where Kine should write its StdOut, StdErr to.
	//
	// If not specified, the output will be discarded.
	Out io.Writer
	Err io.Writer

	ctx    context.Context
	cancel context.CancelFunc
}

// Start starts the etcd, waits for it to come up, and returns an error, if one
// occoured.
func (e *Kine) Start() error {
	e.ctx, e.cancel = context.WithCancel(context.Background())

	listener := ""
	if e.URL != nil {
		listener = e.URL.String()
	}
	config, err := kineendpoint.Listen(e.ctx, kineendpoint.Config{Listener: listener})
	if err != nil {
		return err
	}

	e.URL, err = url.Parse(config.Endpoints[0])
	if err != nil {
		return err
	}
	return nil
}

// Stop stops this process gracefully, waits for its termination, and cleans up
// the DataDir if necessary.
func (e *Kine) Stop() error {
	e.cancel()
	return nil
}

// KineDefaultArgs exposes the default args for Kine so that you
// can use those to append your own additional arguments.
//
// The internal default arguments are explicitly copied here, we don't want to
// allow users to change the internal ones.
var KineDefaultArgs = append([]string{}, internal.KineDefaultArgs...)
