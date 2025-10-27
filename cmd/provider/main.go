/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/rossigee/provider-harbor/apis"
	projectcontroller "github.com/rossigee/provider-harbor/internal/controller/project"
	registrycontroller "github.com/rossigee/provider-harbor/internal/controller/registry"
	scannercontroller "github.com/rossigee/provider-harbor/internal/controller/scanner"
	usercontroller "github.com/rossigee/provider-harbor/internal/controller/user"
	"github.com/rossigee/provider-harbor/internal/version"
)

func main() {
	var (
		app            = kingpin.New(filepath.Base(os.Args[0]), "Native Crossplane provider for Harbor").DefaultEnvars()
		debug          = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod     = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval   = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-harbor"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	// Log startup information with build and configuration details
	log.Info("Provider starting up",
		"provider", "provider-harbor",
		"version", version.Version,
		"go-version", runtime.Version(),
		"platform", runtime.GOOS+"/"+runtime.GOARCH,
		"sync-period", syncPeriod.String(),
		"poll-interval", pollInterval.String(),
		"max-reconcile-rate", *maxReconcileRate,
		"leader-election", *leaderElection,
		"debug-mode", *debug)

	log.Debug("Detailed startup configuration",
		"sync-period", syncPeriod.String(),
		"poll-interval", pollInterval.String(),
		"max-reconcile-rate", *maxReconcileRate)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-harbor",
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	// Add Harbor APIs to scheme
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add Harbor APIs to scheme")

	// Setup native controllers with rate limiting
	o := controller.Options{
		MaxConcurrentReconciles: *maxReconcileRate,
	}

	// Setup Project controller
	kingpin.FatalIfError(projectcontroller.Setup(mgr, o), "Cannot setup Project controller")

	// Setup Scanner controller
	kingpin.FatalIfError(scannercontroller.Setup(mgr, scannercontroller.Options{
		Logger:       log.WithValues("controller", "scanner"),
		PollInterval: pollInterval.String(),
	}), "Cannot setup Scanner controller")

	// Setup User controller
	kingpin.FatalIfError(usercontroller.Setup(mgr, o), "Cannot setup User controller")

	// Setup Registry controller
	kingpin.FatalIfError(registrycontroller.Setup(mgr, o), "Cannot setup Registry controller")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	log.Info("Starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
