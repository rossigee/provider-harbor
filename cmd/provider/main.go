/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/rossigee/provider-harbor/apis"
	membercontroller "github.com/rossigee/provider-harbor/internal/controller/member"
	projectcontroller "github.com/rossigee/provider-harbor/internal/controller/project"
	registrycontroller "github.com/rossigee/provider-harbor/internal/controller/registry"
	replicationcontroller "github.com/rossigee/provider-harbor/internal/controller/replication"
	retentioncontroller "github.com/rossigee/provider-harbor/internal/controller/retention"
	robotcontroller "github.com/rossigee/provider-harbor/internal/controller/robot"
	scannercontroller "github.com/rossigee/provider-harbor/internal/controller/scanner"
	usercontroller "github.com/rossigee/provider-harbor/internal/controller/user"
	usergroupcontroller "github.com/rossigee/provider-harbor/internal/controller/usergroup"
	webhookcontroller "github.com/rossigee/provider-harbor/internal/controller/webhook"
	"github.com/rossigee/provider-harbor/internal/features"
	"github.com/rossigee/provider-harbor/internal/version"
)

func main() {
	var (
		app              = kingpin.New(filepath.Base(os.Args[0]), "Native Crossplane provider for Harbor").DefaultEnvars()
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod       = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval     = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		leaderElection   = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()

		// Management Policies (beta) let an XR opt out of destructive actions via a
		// non-default spec.managementPolicies (e.g. [Observe,Create,Update,LateInitialize]
		// = manage-but-never-delete). On by default in this fork; disable with the flag
		// or ENABLE_MANAGEMENT_POLICIES=false.
		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for Management Policies (honours non-default spec.managementPolicies).").Default("true").Bool()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-harbor"))
	// Always set controller-runtime's root logger. If it is left unset,
	// controller-runtime logs a one-time "log.SetLogger(...) was never called"
	// warning (with a goroutine stack) and silently drops its own framework
	// logs. Verbosity is controlled by the zap level / --debug, not by leaving
	// the logger unset.
	ctrl.SetLogger(zl)

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

	// Build the feature gate set from CLI flags / env.
	feats := &feature.Flags{}
	if *enableManagementPolicies {
		feats.Enable(features.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", features.EnableBetaManagementPolicies)
	}

	// Setup native controllers with rate limiting. Features are carried on the
	// shared controller.Options so every *.Setup(mgr, o) can conditionally enable
	// management-policy support in its reconciler.
	o := xpcontroller.Options{
		MaxConcurrentReconciles: *maxReconcileRate,
		Features:                feats,
	}

	// Setup Project controller
	kingpin.FatalIfError(projectcontroller.Setup(mgr, o), "Cannot setup Project controller")

	// Setup Scanner controller
	kingpin.FatalIfError(scannercontroller.Setup(mgr, scannercontroller.Options{
		Logger:       log.WithValues("controller", "scanner"),
		PollInterval: pollInterval.String(),
		Features:     feats,
	}), "Cannot setup Scanner controller")

	// Setup User controller
	kingpin.FatalIfError(usercontroller.Setup(mgr, o), "Cannot setup User controller")

	// Setup UserGroup controller
	kingpin.FatalIfError(usergroupcontroller.Setup(mgr, o), "Cannot setup UserGroup controller")

	// Setup Registry controller
	kingpin.FatalIfError(registrycontroller.Setup(mgr, o), "Cannot setup Registry controller")

	kingpin.FatalIfError(membercontroller.Setup(mgr, o), "Cannot setup Member controller")



	// Setup Robot controller (Phase 3)
	// The v1beta1 Robot CRD now ships in package/crds and registers in-cluster
	// (since the packaging fix), so the prior "v1beta1 not available" disable no
	// longer holds.
	kingpin.FatalIfError(robotcontroller.Setup(mgr, o), "Cannot setup Robot controller")

	// Setup Webhook controller (Phase 3) — v1beta1 CRD ships and the client now
	// does real CRUD, so enable it (the prior "v1beta1 not available" note is stale).
	kingpin.FatalIfError(webhookcontroller.Setup(mgr, o), "Cannot setup Webhook controller")

	// Setup Replication controller (Phase 4 - Enterprise)
	kingpin.FatalIfError(replicationcontroller.Setup(mgr, o), "Cannot setup Replication controller")

	// Setup Retention controller (Phase 4 - Enterprise)
	kingpin.FatalIfError(retentioncontroller.Setup(mgr, o), "Cannot setup Retention controller")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	log.Info("Starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
