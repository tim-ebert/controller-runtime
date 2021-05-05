package main

import (
	goflag "flag"
	"io/ioutil"
	"os"

	flag "github.com/spf13/pflag"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	crdPaths              = flag.StringSlice("crd-paths", nil, "paths to files or directories containing CRDs to install on start")
	webhookPaths          = flag.StringSlice("webhook-paths", nil, "paths to files or directories containing webhook configurations to install on start")
	attachControlPlaneOut = flag.Bool("debug-env", false, "attach to test env (apiserver & etcd) output -- just a convinience flag to force KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true")
)

// have a separate function so we can return an exit code w/o skipping defers
func runMain() int {
	loggerOpts := &zap.Options{
		Development: true, // a sane default
	}
	{
		var goFlagSet goflag.FlagSet
		loggerOpts.BindFlags(&goFlagSet)
		flag.CommandLine.AddGoFlagSet(&goFlagSet)
	}
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(loggerOpts)))

	log := ctrl.Log.WithName("main")

	env := &envtest.Environment{}
	env.CRDInstallOptions.Paths = *crdPaths
	env.WebhookInstallOptions.Paths = *webhookPaths

	if *attachControlPlaneOut {
		os.Setenv("KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT", "true")
	}

	log.Info("Starting apiserver & etcd")
	cfg, err := env.Start()
	if err != nil {
		log.Error(err, "unable to start the test environment")
		// shut down the environment in case we started it and failed while
		// installing CRDs or provisioning users.
		if err := env.Stop(); err != nil {
			log.Error(err, "unable to stop the test environment after an error (this might be expected, but just though you should know)")
		}
		return 1
	}

	log.Info("apiserver running", "host", cfg.Host)

	// NB(directxman12): this group is unfortunately named, but various
	// kubernetes versions require us to use it to get "admin" access.
	user, err := env.ControlPlane.AddUser(envtest.User{
		Name:   "envtest-admin",
		Groups: []string{"system:masters"},
	}, nil)
	if err != nil {
		log.Error(err, "unable to provision admin user, continuing on without it")
	} else {
		// TODO(directxman12): add support for writing to a new context in an existing file
		kubeconfigFile, err := ioutil.TempFile("", "scratch-env-kubeconfig-")
		if err != nil {
			log.Error(err, "unable to create kubeconfig file, continuing on without it")
		} else {
			defer os.Remove(kubeconfigFile.Name())

			log := log.WithValues("path", kubeconfigFile.Name())
			log.V(1).Info("Writing kubeconfig")

			kubeConfig, err := user.KubeConfig()
			if err != nil {
				log.Error(err, "unable to create kubeconfig")
			}

			if _, err := kubeconfigFile.Write(kubeConfig); err != nil {
				log.Error(err, "unable to save kubeconfig")
				return 1
			}

			log.Info("Wrote kubeconfig")
		}
	}

	ctx := ctrl.SetupSignalHandler()
	<-ctx.Done()

	log.Info("Shutting down apiserver & etcd")
	err = env.Stop()
	if err != nil {
		log.Error(err, "unable to stop the test environment")
		return 1
	}

	log.Info("Shutdown succesful")
	return 0
}

func main() {
	os.Exit(runMain())
}
