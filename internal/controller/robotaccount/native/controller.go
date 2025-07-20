package native

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/globallogicuki/provider-harbor/apis/robotaccount/v1alpha1"
	apisv1beta1 "github.com/globallogicuki/provider-harbor/apis/v1beta1"
)

const (
	errNotRobotAccount = "managed resource is not a RobotAccount custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errCreateClient    = "cannot create Harbor client"
)

// Setup adds a controller that reconciles RobotAccount managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RobotAccount_GroupVersionKind.String())

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RobotAccount_GroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1beta1.ProviderConfigUsage{}),
			newServiceFn: newHarborClient,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.RobotAccount{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(ctx context.Context, endpoint, username, password string, insecure bool) (*HarborClient, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.RobotAccount)
	if !ok {
		return nil, errors.New(errNotRobotAccount)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	// Parse credentials
	var creds struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
		Insecure bool   `json:"insecure,omitempty"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal credentials")
	}

	svc, err := c.newServiceFn(ctx, creds.URL, creds.Username, creds.Password, creds.Insecure)
	if err != nil {
		return nil, errors.Wrap(err, errCreateClient)
	}

	return &external{client: svc, kube: c.kube}, nil
}

// HarborClientInterface defines the interface for Harbor operations
type HarborClientInterface interface {
	CreateRobotAccount(spec RobotAccountSpec) (*models.Robot, error)
	GetRobotAccount(robotID int64) (*models.Robot, error)
	GetRobotAccountByName(name string) (*models.Robot, error)
	DeleteRobotAccount(robotID int64) error
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client HarborClientInterface
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.RobotAccount)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobotAccount)
	}

	// If we have an external name, try to get the robot account by ID
	if meta.GetExternalName(cr) != "" {
		robotID, err := ExtractRobotID(meta.GetExternalName(cr))
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "cannot extract robot ID from external name")
		}

		robot, err := e.client.GetRobotAccount(robotID)
		if err != nil {
			// If not found, the resource doesn't exist
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		// Update observation - convert from models.Robot to observation
		robotIDStr := strconv.FormatInt(robot.ID, 10)
		cr.Status.AtProvider = v1alpha1.RobotAccountObservation{
			ID:       &robotIDStr,
			FullName: &robot.Name,
			RobotID:  &robotIDStr,
			Disable:  &robot.Disable,
			Level:    &robot.Level,
		}

		// Check if the resource is up to date
		upToDate := isRobotAccountUpToDate(cr.Spec.ForProvider, robot)

		return managed.ExternalObservation{
			ResourceExists:    true,
			ResourceUpToDate:  upToDate,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	// No external name, try to find by name
	name := *cr.Spec.ForProvider.Name
	robot, err := e.client.GetRobotAccountByName(name)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Set external name
	meta.SetExternalName(cr, fmt.Sprintf("/robots/%d", robot.ID))

	// Update observation - convert from models.Robot to observation
	robotIDStr := strconv.FormatInt(robot.ID, 10)
	cr.Status.AtProvider = v1alpha1.RobotAccountObservation{
		ID:       &robotIDStr,
		FullName: &robot.Name,
		RobotID:  &robotIDStr,
		Disable:  &robot.Disable,
		Level:    &robot.Level,
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isRobotAccountUpToDate(cr.Spec.ForProvider, robot),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.RobotAccount)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobotAccount)
	}

	// Convert CR spec to client spec
	spec := RobotAccountSpec{
		Name:        *cr.Spec.ForProvider.Name,
		Description: valueOrDefault(cr.Spec.ForProvider.Description, ""),
		Duration:    int64(valueOrDefault(cr.Spec.ForProvider.Duration, float64(-1))),
		Level:       valueOrDefault(cr.Spec.ForProvider.Level, "project"),
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	robot, err := e.client.CreateRobotAccount(spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create robot account")
	}

	// Set external name
	meta.SetExternalName(cr, fmt.Sprintf("/robots/%d", robot.ID))

	// Store the secret
	conn := managed.ConnectionDetails{}
	if robot.Secret != "" {
		conn[xpv1.ResourceCredentialsSecretPasswordKey] = []byte(robot.Secret)
		conn[xpv1.ResourceCredentialsSecretUserKey] = []byte(robot.Name)
	}

	// Connection secret is managed by the ConnectionPublisher in managed reconciler
	// We only need to return the connection details

	return managed.ExternalCreation{
		ConnectionDetails: conn,
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// Harbor doesn't support updating robot accounts
	// We would need to delete and recreate
	return managed.ExternalUpdate{}, errors.New("robot account updates are not supported")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.RobotAccount)
	if !ok {
		return errors.New(errNotRobotAccount)
	}

	if meta.GetExternalName(cr) == "" {
		// No external name means the resource was never created
		return nil
	}

	robotID, err := ExtractRobotID(meta.GetExternalName(cr))
	if err != nil {
		return errors.Wrap(err, "cannot extract robot ID from external name")
	}

	if err := e.client.DeleteRobotAccount(robotID); err != nil {
		return errors.Wrap(err, "cannot delete robot account")
	}

	// Connection secret is managed by the ConnectionPublisher in managed reconciler

	return nil
}

// Helper functions

func isRobotAccountUpToDate(spec v1alpha1.RobotAccountParameters, robot *models.Robot) bool {
	// Harbor doesn't allow updates, so we only check if the basic properties match
	// Harbor returns robot names with prefix like "robot$project+name" or "robot$system+name"
	// We need to extract the suffix for comparison
	if spec.Name != nil {
		expectedPrefix := "robot$"
		if spec.Level != nil {
			expectedPrefix = fmt.Sprintf("robot$%s+", *spec.Level)
		}
		expectedName := expectedPrefix + *spec.Name
		if !strings.HasPrefix(robot.Name, expectedPrefix) || robot.Name != expectedName {
			return false
		}
	}
	if spec.Level != nil && *spec.Level != robot.Level {
		return false
	}
	// We can't compare permissions as Harbor doesn't return them in GET
	return true
}

func convertPermissions(perms []v1alpha1.PermissionsParameters) []Permission {
	result := make([]Permission, len(perms))
	for i, p := range perms {
		access := make([]Access, len(p.Access))
		for j, a := range p.Access {
			access[j] = Access{
				Resource: *a.Resource,
				Action:   *a.Action,
			}
		}
		result[i] = Permission{
			Kind:      *p.Kind,
			Namespace: *p.Namespace,
			Access:    access,
		}
	}
	return result
}

func valueOrDefault[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}

// newHarborClient is a helper function for creating a Harbor client
var newHarborClient = NewHarborClient