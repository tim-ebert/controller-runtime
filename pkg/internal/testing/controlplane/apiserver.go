package controlplane

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/internal/testing/addr"
	"sigs.k8s.io/controller-runtime/pkg/internal/testing/certs"
	"sigs.k8s.io/controller-runtime/pkg/internal/testing/process"
)

// APIServer knows how to run a kubernetes apiserver.
type APIServer struct {
	// URL is the address the ApiServer should listen on for client connections.
	//
	// If this is not specified, we default to a random free port on localhost.
	URL *url.URL

	// SecurePort is the additional secure port that the APIServer should listen on.
	SecurePort int

	// Path is the path to the apiserver binary.
	//
	// If this is left as the empty string, we will attempt to locate a binary,
	// by checking for the TEST_ASSET_KUBE_APISERVER environment variable, and
	// the default test assets directory. See the "Binaries" section above (in
	// doc.go) for details.
	Path string

	// Args is a list of arguments which will passed to the APIServer binary.
	// Before they are passed on, they will be evaluated as go-template strings.
	// This means you can use fields which are defined and exported on this
	// APIServer struct (e.g. "--cert-dir={{ .Dir }}").
	// Those templates will be evaluated after the defaulting of the APIServer's
	// fields has already happened and just before the binary actually gets
	// started. Thus you have access to calculated fields like `URL` and others.
	//
	// If not specified, the minimal set of arguments to run the APIServer will
	// be used.
	//
	// They will be loaded into the same argument set as Configure.  Each flag
	// will be Append-ed to the configured arguments just before launch.
	//
	// Deprecated: use Configure instead.
	Args []string

	// CertDir is a path to a directory containing whatever certificates the
	// APIServer will need.
	//
	// If left unspecified, then the Start() method will create a fresh temporary
	// directory, and the Stop() method will clean it up.
	CertDir string

	// EtcdURL is the URL of the Etcd the APIServer should use.
	//
	// If this is not specified, the Start() method will return an error.
	EtcdURL *url.URL

	// StartTimeout, StopTimeout specify the time the APIServer is allowed to
	// take when starting and stoppping before an error is emitted.
	//
	// If not specified, these default to 20 seconds.
	StartTimeout time.Duration
	StopTimeout  time.Duration

	// Out, Err specify where APIServer should write its StdOut, StdErr to.
	//
	// If not specified, the output will be discarded.
	Out io.Writer
	Err io.Writer

	processState *process.State

	// args contains the structured arguments to use for running the API server
	// Lazily initialized by .Configure(), Defaulted eventually with .defaultArgs()
	args *process.Arguments
}

// Configure returns Arguments that may be used to customize the
// flags used to launch the API server.  A set of defaults will
// be applied underneath.
func (s *APIServer) Configure() *process.Arguments {
	if s.args == nil {
		s.args = process.EmptyArguments()
	}
	return s.args
}

// Start starts the apiserver, waits for it to come up, and returns an error,
// if occurred.
func (s *APIServer) Start() error {
	if s.processState == nil {
		if err := s.setProcessState(); err != nil {
			return err
		}
	}
	return s.processState.Start(s.Out, s.Err)
}

func (s *APIServer) setProcessState() error {
	if s.EtcdURL == nil {
		return fmt.Errorf("expected EtcdURL to be configured")
	}

	var err error

	s.processState = &process.State{
		Dir:          s.CertDir,
		Path:         s.Path,
		StartTimeout: s.StartTimeout,
		StopTimeout:  s.StopTimeout,
	}
	if err := s.processState.Init("kube-apiserver"); err != nil {
		return err
	}

	// Defaulting the secure port
	if s.SecurePort == 0 {
		s.SecurePort, _, err = addr.Suggest("")
		if err != nil {
			return err
		}
	}

	if s.URL == nil {
		port, host, err := addr.Suggest("")
		if err != nil {
			return err
		}
		s.URL = &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		}
	}

	s.processState.HealthCheck.URL = *s.URL
	s.processState.HealthCheck.Path = "/healthz"

	s.CertDir = s.processState.Dir
	s.Path = s.processState.Path
	s.StartTimeout = s.processState.StartTimeout
	s.StopTimeout = s.processState.StopTimeout

	if err := s.populateAPIServerCerts(); err != nil {
		return err
	}

	s.processState.Args, err = process.TemplateAndArguments(s.Args, s.Configure(), process.TemplateDefaults{
		Data:     s,
		Defaults: s.defaultArgs(),
		// as per kubernetes-sigs/controller-runtime#641, we need this (we
		// probably need other stuff too, but this is the only thing that was
		// previously considered a "minimal default")
		MinimalDefaults: map[string][]string{
			"service-cluster-ip-range": []string{"10.0.0.0/24"},
		},
	})
	return err
}

func (s *APIServer) defaultArgs() map[string][]string {
	args := map[string][]string{
		"advertise-address":        []string{"127.0.0.1"},
		"service-cluster-ip-range": []string{"10.0.0.0/24"},
		"allow-privileged":         []string{"true"},
		// we're keeping this disabled because if enabled, default SA is
		// missing which would force all tests to create one in normal
		// apiserver operation this SA is created by controller, but that is
		// not run in integration environment
		"disable-admission-plugins": []string{"ServiceAccount"},
	}
	if s.EtcdURL != nil {
		args["etcd-servers"] = []string{s.EtcdURL.String()}
	}
	if s.CertDir != "" {
		args["cert-dir"] = []string{s.CertDir}
	}
	if s.URL != nil {
		args["insecure-port"] = []string{s.URL.Port()}
		args["insecure-bind-address"] = []string{s.URL.Hostname()}
	}
	return args
}

func (s *APIServer) populateAPIServerCerts() error {
	_, statErr := os.Stat(filepath.Join(s.CertDir, "apiserver.crt"))
	if !os.IsNotExist(statErr) {
		return statErr
	}

	ca, err := certs.NewTinyCA()
	if err != nil {
		return err
	}

	certs, err := ca.NewServingCert()
	if err != nil {
		return err
	}

	certData, keyData, err := certs.AsBytes()
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(s.CertDir, "apiserver-ca.crt"), ca.CA.CertBytes(), 0640); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(s.CertDir, "apiserver.crt"), certData, 0640); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(s.CertDir, "apiserver.key"), keyData, 0640); err != nil {
		return err
	}

	return nil
}

// Stop stops this process gracefully, waits for its termination, and cleans up
// the CertDir if necessary.
func (s *APIServer) Stop() error {
	if s.processState != nil {
		return s.processState.Stop()
	}
	return nil
}

// APIServerDefaultArgs exposes the default args for the APIServer so that you
// can use those to append your own additional arguments.
var APIServerDefaultArgs = []string{
	"--advertise-address=127.0.0.1",
	"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
	"--cert-dir={{ .CertDir }}",
	"--insecure-port={{ if .URL }}{{ .URL.Port }}{{ end }}",
	"--insecure-bind-address={{ if .URL }}{{ .URL.Hostname }}{{ end }}",
	"--secure-port={{ if .SecurePort }}{{ .SecurePort }}{{ end }}",
	// we're keeping this disabled because if enabled, default SA is missing which would force all tests to create one
	// in normal apiserver operation this SA is created by controller, but that is not run in integration environment
	"--disable-admission-plugins=ServiceAccount",
	"--service-cluster-ip-range=10.0.0.0/24",
	"--allow-privileged=true",
}
