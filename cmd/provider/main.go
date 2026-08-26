/*
Copyright 2024 Crossplane Harbor Provider.
*/

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/rossigee/provider-harbor/apis"
	artifactv1beta1 "github.com/rossigee/provider-harbor/apis/artifact/v1beta1"
	memberv1beta1 "github.com/rossigee/provider-harbor/apis/member/v1beta1"
	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	registryv1beta1 "github.com/rossigee/provider-harbor/apis/registry/v1beta1"
	replicationv1beta1 "github.com/rossigee/provider-harbor/apis/replication/v1beta1"
	repositoryv1beta1 "github.com/rossigee/provider-harbor/apis/repository/v1beta1"
	retentionv1beta1 "github.com/rossigee/provider-harbor/apis/retention/v1beta1"
	robotv1beta1 "github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	scanv1beta1 "github.com/rossigee/provider-harbor/apis/scan/v1beta1"
	scannerv1beta1 "github.com/rossigee/provider-harbor/apis/scanner/v1beta1"
	userv1beta1 "github.com/rossigee/provider-harbor/apis/user/v1beta1"
	usergroupv1beta1 "github.com/rossigee/provider-harbor/apis/usergroup/v1beta1"
	webhookv1beta1 "github.com/rossigee/provider-harbor/apis/webhook/v1beta1"
	harborcontroller "github.com/rossigee/provider-harbor/internal/controller"
	artifactcontroller "github.com/rossigee/provider-harbor/internal/controller/artifact"
	membercontroller "github.com/rossigee/provider-harbor/internal/controller/member"
	projectcontroller "github.com/rossigee/provider-harbor/internal/controller/project"
	registrycontroller "github.com/rossigee/provider-harbor/internal/controller/registry"
	replicationcontroller "github.com/rossigee/provider-harbor/internal/controller/replication"
	repositorycontroller "github.com/rossigee/provider-harbor/internal/controller/repository"
	retentioncontroller "github.com/rossigee/provider-harbor/internal/controller/retention"
	robotcontroller "github.com/rossigee/provider-harbor/internal/controller/robot"
	scancontroller "github.com/rossigee/provider-harbor/internal/controller/scan"
	scannercontroller "github.com/rossigee/provider-harbor/internal/controller/scanner"
	usercontroller "github.com/rossigee/provider-harbor/internal/controller/user"
	usergroupcontroller "github.com/rossigee/provider-harbor/internal/controller/usergroup"
	webhookcontroller "github.com/rossigee/provider-harbor/internal/controller/webhook"
	"github.com/rossigee/provider-harbor/internal/tracing"
	"github.com/rossigee/provider-harbor/internal/version"
	"gopkg.in/alecthomas/kingpin.v2"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	// Enable controller-runtime debug logging
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("CATTLE_DEVELOPER_LOGGING", "true")
	var (
		app                     = kingpin.New(filepath.Base(os.Args[0]), "Native Crossplane provider for Harbor").DefaultEnvars()
		debug                   = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod              = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval            = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		leaderElection          = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate        = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
		pollStateMetricInterval = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		metricsBindAddress      = app.Flag("metrics-bind-address", "The address the metrics endpoint binds to.").Default(":8080").String()
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

	s := apimachineryruntime.NewScheme()
	kingpin.FatalIfError(scheme.AddToScheme(s), "Cannot add k8s types to scheme")
	kingpin.FatalIfError(apis.AddToScheme(s), "Cannot add Harbor APIs to scheme")

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           s,
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-harbor",
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
		Metrics: metricserver.Options{
			BindAddress: *metricsBindAddress,
		},
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	mrStateMetrics := statemetrics.NewMRStateMetrics()
	metrics.Registry.MustRegister(mrStateMetrics)

	// Setup native controllers with rate limiting
	o := controller.Options{
		MaxConcurrentReconciles: *maxReconcileRate,
	}

	kingpin.FatalIfError(harborcontroller.Setup(mgr), "Cannot setup Harbor controllers")

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

	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &projectv1beta1.ProjectList{}, *pollStateMetricInterval)), "Cannot register state metrics for Project")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &registryv1beta1.RegistryList{}, *pollStateMetricInterval)), "Cannot register state metrics for Registry")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &repositoryv1beta1.RepositoryList{}, *pollStateMetricInterval)), "Cannot register state metrics for Repository")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &artifactv1beta1.ArtifactList{}, *pollStateMetricInterval)), "Cannot register state metrics for Artifact")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &memberv1beta1.MemberList{}, *pollStateMetricInterval)), "Cannot register state metrics for Member")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &scanv1beta1.ScanList{}, *pollStateMetricInterval)), "Cannot register state metrics for Scan")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &robotv1beta1.RobotList{}, *pollStateMetricInterval)), "Cannot register state metrics for Robot")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &userv1beta1.UserList{}, *pollStateMetricInterval)), "Cannot register state metrics for User")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &usergroupv1beta1.UserGroupList{}, *pollStateMetricInterval)), "Cannot register state metrics for UserGroup")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &scannerv1beta1.ScannerRegistrationList{}, *pollStateMetricInterval)), "Cannot register state metrics for ScannerRegistration")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &webhookv1beta1.WebhookList{}, *pollStateMetricInterval)), "Cannot register state metrics for Webhook")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &replicationv1beta1.ReplicationList{}, *pollStateMetricInterval)), "Cannot register state metrics for Replication")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), log, mrStateMetrics, &retentionv1beta1.RetentionList{}, *pollStateMetricInterval)), "Cannot register state metrics for Retention")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	log.Info("All controllers initialized, starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
