package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/operators/catalog"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/version"
)

const (
	defaultWakeupInterval   = 15 * time.Minute
	defaultCatalogNamespace = "tectonic-system"
)

// config flags defined globally so that they appear on the test binary as well
var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "absolute path to the kubeconfig file")

	wakeupInterval = flag.Duration(
		"interval", defaultWakeupInterval, "wakeup interval")

	watchedNamespaces = flag.String(
		"watchedNamespaces", "", "comma separated list of namespaces that catalog watches, leave empty to watch all namespaces")

	catalogNamespace = flag.String(
		"namespace", defaultCatalogNamespace, "namespace where catalog will run and install catalog resources")

	debug = flag.Bool(
		"debug", false, "use debug log level")

	version = flag.Bool("version", false, "displays olm version")
)

func main() {
	stopCh := signals.SetupSignalHandler()

	// Parse the command-line flags.
	flag.Parse()

	// Check if version flag was set
	if *version {
		fmt.Print(olmversion.String())

		// Exit early
		os.Exit(0)
	}

	// `namespaces` will always contain at least one entry: if `*watchedNamespaces` is
	// the empty string, the resulting array will be `[]string{""}`.
	namespaces := strings.Split(*watchedNamespaces, ",")
	for _, ns := range namespaces {
		if ns == v1.NamespaceAll {
			namespaces = []string{v1.NamespaceAll}
			break
		}
	}

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	logger := log.New()
	if *debug {
		logger.SetLevel(log.DebugLevel)
	}
	logger.Infof("log level %s", logger.Level)

	// Create a new instance of the operator.
	catalogOperator, err := catalog.NewOperator(*kubeConfigPath, logger, *wakeupInterval, *catalogNamespace, namespaces...)
	if err != nil {
		log.Panicf("error configuring operator: %s", err.Error())
	}

	_, done := catalogOperator.Run(stopCh)
	<-done
}
