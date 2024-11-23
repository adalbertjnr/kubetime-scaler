package scheduler

import (
	"context"
	"fmt"
	"time"

	downscalergov1alpha1 "github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	"github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/factory"
	"github.com/adalbertjnr/downscalerk8s/internal/store"
	"github.com/adalbertjnr/downscalerk8s/internal/types"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Downscaler struct {
	app downscalergov1alpha1.Downscaler

	client *client.APIClient

	cron *cron.Cron

	log logr.Logger

	getFactory *factory.FactoryScaler

	downscalerNamespace bool

	store       *store.Persistence
	persistence bool

	cancelFunc context.CancelFunc
}

func (dc *Downscaler) Client(c *client.APIClient) *Downscaler {
	dc.client = c
	return dc
}

func (dc *Downscaler) Factory(f *factory.FactoryScaler) *Downscaler {
	dc.getFactory = f
	return dc
}

func (dc *Downscaler) Persistence(p *store.Persistence) *Downscaler {
	if p != nil {
		dc.store = p
		dc.persistence = true
	}
	return dc
}

func (dc *Downscaler) Logger(l logr.Logger) *Downscaler {
	dc.log = l
	return dc
}

func (dc *Downscaler) handleDatabase() {
	if !dc.persistence {
		return
	}
	if err := dc.store.ScalingOperation.Bootstrap(context.Background()); err != nil {
		dc.log.Error(err, "database", "creation error", err)
		return
	}
}

func (dc *Downscaler) Run() (ctrl.Result, error) {
	if err := dc.resetState().createNewClient(); err != nil {
		return ctrl.Result{}, err
	}

	dc.handleDatabase()

	dc.initializeCronTasks()

	return ctrl.Result{}, nil
}

func (dc *Downscaler) addCronJob(scaleStr string, namespace downscalergov1alpha1.Namespace, defaultScaleReplicas types.ScalingOperation) {
	expression := dc.buildCronExpression(dc.recurrence(), scaleStr)

	entryID, err := dc.cron.AddFunc(expression, dc.job(namespace, defaultScaleReplicas))
	if err != nil {
		dc.log.Error(err, "cron", "scheduling error", err)
		return
	}

	dc.log.Info("cron", "namespace", namespace, "assigning new cron entryID", entryID)
}

func (dc *Downscaler) job(namespace downscalergov1alpha1.Namespace, defaultScaleReplicas types.ScalingOperation) func() {
	return func() {
		for _, rule := range dc.rules() {
			if namespace.Found(rule.Namespaces) {

				overrideResource := rule.OverrideScaling
				if len(overrideResource) == 0 {
					overrideResource = dc.resourceScaling()
				}

				dc.execute(namespace.String(), defaultScaleReplicas, overrideResource)
			}
		}
	}
}

func (dc *Downscaler) execute(namespace string, replicas types.ScalingOperation, overrideResource []string) {
	for _, resource := range overrideResource {
		if resourceScaler, created := (*dc.getFactory)[resource]; created {
			if err := resourceScaler.Run(namespace, replicas); err != nil {
				dc.log.Error(err, "job", "resource", resource, "scaling error", err)
				return
			}
		}
	}
}

func (dc *Downscaler) initializeCronTasks() {
	for _, rule := range dc.rules() {
		for _, namespace := range rule.Namespaces {
			dc.addCronJob(rule.UpscaleTime, namespace, types.OperationUpscale)
			dc.addCronJob(rule.DownscaleTime, namespace, types.OperationDownscale)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	dc.cancelFunc = cancel

	go dc.notifyCronEntries(ctx)

	dc.cron.Start()
}

func (dc *Downscaler) notifyCronEntries(ctx context.Context) {
	interval := dc.app.Spec.Config.CronLoggerInterval
	if interval <= 0 {
		interval = 300
	}

	ticker := time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, entry := range dc.cron.Entries() {
				dc.log.Info("cron", "entryID", entry.ID, "nextRun", entry.Next)
			}
		}
	}
}

func (s *Downscaler) Add(ctx context.Context, app downscalergov1alpha1.Downscaler) *Downscaler {
	s.app = app
	return s
}

func (dc *Downscaler) resetState() *Downscaler {
	if dc.cron != nil {
		dc.cron.Stop()
		dc.cron = nil
	}
	if dc.cancelFunc != nil {
		dc.cancelFunc()
	}
	return dc
}

func (dc *Downscaler) createNewClient() error {
	if dc.cron == nil {
		downscaler, err := dc.client.GetDownscaler()
		if err != nil {
			return fmt.Errorf("error getting downscaler object: %v", err)
		}

		location, err := time.LoadLocation(downscaler.Spec.Schedule.TimeZone)
		if err != nil {
			return fmt.Errorf("error loading object timezone: %v", err)
		}

		cron := cron.New(cron.WithLocation(location))
		dc.cron = cron
	}

	return nil
}

func (dc *Downscaler) buildCronExpression(recurrence, timeStr string) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		dc.log.Error(err, "Invalid time format", "timeStr", timeStr, "error", err)
		return "0 0 * * *"
	}

	if recurrence == "*" || recurrence == "@daily" {
		return fmt.Sprintf("%d %d * * *", t.Minute(), t.Hour())
	}

	return fmt.Sprintf("%d %d * * %s", t.Minute(), t.Hour(), recurrence)
}

func (dc *Downscaler) namespaces() []string {
	var namespaceList []string
	for _, rule := range dc.rules() {
		for _, namespace := range rule.Namespaces {
			namespaceList = append(namespaceList, string(namespace))
		}
	}

	return namespaceList
}

func (dc *Downscaler) rules() []downscalergov1alpha1.Rules {
	return dc.app.Spec.DownscalerOptions.TimeRules.Rules
}

func (dc *Downscaler) resourceScaling() []string {
	return dc.app.Spec.DownscalerOptions.ResourceScaling
}

func (dc *Downscaler) recurrence() string {
	return dc.app.Spec.Schedule.Recurrence
}
