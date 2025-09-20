/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/rossigee/provider-harbor/apis"
	projectcontroller "github.com/rossigee/provider-harbor/internal/controller/project"
	scannercontroller "github.com/rossigee/provider-harbor/internal/controller/scanner"
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

	log.Info("Native Harbor provider starting")
	log.Debug("Starting", "sync-period", syncPeriod.String(), "poll-interval", pollInterval.String(), "max-reconcile-rate", *maxReconcileRate)

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

	log.Info("Starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
