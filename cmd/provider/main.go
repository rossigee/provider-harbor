/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"context"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/rossigee/provider-harbor/apis"
	"github.com/rossigee/provider-harbor/internal/controller/artifact"
	"github.com/rossigee/provider-harbor/internal/controller/member"
	"github.com/rossigee/provider-harbor/internal/controller/project"
	"github.com/rossigee/provider-harbor/internal/controller/registry"
	"github.com/rossigee/provider-harbor/internal/controller/replication"
	"github.com/rossigee/provider-harbor/internal/controller/repository"
	"github.com/rossigee/provider-harbor/internal/controller/retention"
	"github.com/rossigee/provider-harbor/internal/controller/robot"
	"github.com/rossigee/provider-harbor/internal/controller/scan"
	"github.com/rossigee/provider-harbor/internal/controller/scanner"
	"github.com/rossigee/provider-harbor/internal/controller/user"
	"github.com/rossigee/provider-harbor/internal/controller/usergroup"
	"github.com/rossigee/provider-harbor/internal/controller/webhook"
	"github.com/rossigee/provider-harbor/internal/tracing"
	"github.com/rossigee/provider-harbor/internal/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"path/filepath"
	"runtime"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

func main() {
	// Enable controller-runtime debug logging
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("CATTLE_DEVELOPER_LOGGING", "true")
	var (
		app              = kingpin.New(filepath.Base(os.Args[0]), "Native Crossplane provider for Harbor").DefaultEnvars()
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod       = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval     = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		leaderElection   = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	ctrl.SetLogger(zl)
	crlog.SetLogger(zl)
	log := logging.NewLogrLogger(zl.WithName("provider-harbor"))

	shutdownTracing := tracing.Init("provider-harbor")
	defer shutdownTracing(context.Background())

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

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-harbor",
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
			DefaultNamespaces: map[string]cache.Config{
				"crossplane-system": {},
				"harbor-projects":   {},
			},
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

	// Setup Registry controller
	kingpin.FatalIfError(registrycontroller.Setup(mgr, o), "Cannot setup Registry controller")

	// Setup Repository controller
	kingpin.FatalIfError(repositorycontroller.Setup(mgr, o), "Cannot setup Repository controller")

	// Setup Artifact controller
	kingpin.FatalIfError(artifactcontroller.Setup(mgr, o), "Cannot setup Artifact controller")

	// Setup Member controller
	kingpin.FatalIfError(membercontroller.Setup(mgr, o), "Cannot setup Member controller")

	// Setup Scan controller
	kingpin.FatalIfError(scancontroller.Setup(mgr, o), "Cannot setup Scan controller")

	// Setup Robot controller
	kingpin.FatalIfError(robotcontroller.Setup(mgr, o), "Cannot setup Robot controller")

	// Setup User controller
	kingpin.FatalIfError(usercontroller.Setup(mgr, o), "Cannot setup User controller")

	// Setup UserGroup controller
	kingpin.FatalIfError(usergroupcontroller.Setup(mgr, o), "Cannot setup UserGroup controller")

	// Setup Scanner controller
	kingpin.FatalIfError(scannercontroller.Setup(mgr, o), "Cannot setup Scanner controller")

	// Setup Webhook controller
	kingpin.FatalIfError(webhookcontroller.Setup(mgr, o), "Cannot setup Webhook controller")

	// Setup Replication controller
	kingpin.FatalIfError(replicationcontroller.Setup(mgr, o), "Cannot setup Replication controller")

	// Setup Retention controller
	kingpin.FatalIfError(retentioncontroller.Setup(mgr, o), "Cannot setup Retention controller")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	log.Info("All controllers initialized, starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
