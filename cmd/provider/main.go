/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	// replicationcontroller "github.com/rossigee/provider-harbor/internal/controller/replication"
	// repositorycontroller "github.com/rossigee/provider-harbor/internal/controller/repository"
	// retentioncontroller "github.com/rossigee/provider-harbor/internal/controller/retention"
	// robotcontroller "github.com/rossigee/provider-harbor/internal/controller/robot"
	// scancontroller "github.com/rossigee/provider-harbor/internal/controller/scan"
	// scannercontroller "github.com/rossigee/provider-harbor/internal/controller/scanner"
	// artifactcontroller "github.com/rossigee/provider-harbor/internal/controller/artifact"
	// membercontroller "github.com/rossigee/provider-harbor/internal/controller/member"
	// usercontroller "github.com/rossigee/provider-harbor/internal/controller/user"
	// usergroupcontroller "github.com/rossigee/provider-harbor/internal/controller/usergroup"
	// webhookcontroller "github.com/rossigee/provider-harbor/internal/controller/webhook"
	"github.com/rossigee/provider-harbor/internal/version"
)

func main() {
	os.Stderr.WriteString("DEBUG: Provider main() started\n")
	
	// Enable controller-runtime debug logging  
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("CATTLE_DEVELOPER_LOGGING", "true")
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
	log := logging.NewLogrLogger(zl.WithName("provider-harbor"))
	// Always set the logger - this is needed for proper debug output
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
	os.Stderr.WriteString("DEBUG: Controller manager created successfully\n")

	// Add Harbor APIs to scheme
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add Harbor APIs to scheme")
	os.Stderr.WriteString("DEBUG: APIs added to scheme\n")
	
	// Check if Project type is registered
	scheme := mgr.GetScheme()
	if scheme != nil {
		os.Stderr.WriteString("DEBUG: Scheme has types: ")
		types := scheme.AllKnownTypes()
		os.Stderr.WriteString(fmt.Sprintf("Found %d types\n", len(types)))
		for k := range types {
			if strings.Contains(k.Kind, "Project") {
				os.Stderr.WriteString("DEBUG: Found Project type: " + k.String() + "\n")
			}
		}
	}

	// Setup native controllers with rate limiting
	o := controller.Options{
		MaxConcurrentReconciles: *maxReconcileRate,
	}

	// Setup Project controller
	if err := projectcontroller.Setup(mgr, o); err != nil {
		os.Stderr.WriteString("ERROR: Failed to setup Project controller: " + err.Error() + "\n")
		kingpin.FatalIfError(err, "Cannot setup Project controller")
	}
	os.Stderr.WriteString("DEBUG: Project controller setup completed\n")

	// Setup Scanner controller - DISABLED (cache sync timeout)
	// Setup User controller - DISABLED (cache sync timeout)
	// Setup UserGroup controller - DISABLED (cache sync timeout)

	// Setup Registry controller
	kingpin.FatalIfError(registrycontroller.Setup(mgr, o), "Cannot setup Registry controller")

	// Setup Repository controller - DISABLED (cache sync timeout)
	// Setup Artifact controller - DISABLED (cache sync timeout)
	// Setup Member controller - DISABLED (cache sync timeout)
	// Setup Scan controller - DISABLED (cache sync timeout)

	// Setup Robot controller - DISABLED (cache sync timeout bugs)
	// TODO: Fix Robot controller implementation - has issues with cache synchronization
	// kingpin.FatalIfError(robotcontroller.Setup(mgr, o), "Cannot setup Robot controller")
	// os.Stderr.WriteString("DEBUG: Robot controller setup completed\n")

	// Setup Webhook controller (Phase 3)
	// DISABLED: CRD v1beta1 not available in cluster (only v1alpha1 exists)
	// kingpin.FatalIfError(webhookcontroller.Setup(mgr, o), "Cannot setup Webhook controller")

	// Setup Replication controller (Phase 4 - Enterprise)
	// DISABLED: CRD v1beta1 not available in cluster (only v1alpha1 exists)
	// kingpin.FatalIfError(replicationcontroller.Setup(mgr, o), "Cannot setup Replication controller")

	// Setup Retention controller (Phase 4 - Enterprise)
	// DISABLED: CRD v1beta1 not available in cluster (only v1alpha1 exists)
	// kingpin.FatalIfError(retentioncontroller.Setup(mgr, o), "Cannot setup Retention controller")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	log.Info("All controllers initialized, starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
