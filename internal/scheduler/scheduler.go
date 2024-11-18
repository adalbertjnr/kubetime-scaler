package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	downscalergov1alpha1 "github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	"github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/factory"
	"github.com/adalbertjnr/downscalerk8s/internal/store"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	downscaleReplicas int = 0
	upscaleReplicas   int = 1
)

type Downscaler struct {
	app downscalergov1alpha1.Downscaler

	log logr.Logger

	client *client.APIClient

	cron *cron.Cron

	downscalerNamespace bool

	store       *store.Persistence
	persistence bool

	cancelFunc context.CancelFunc
}

func (dc *Downscaler) Client(c *client.APIClient) *Downscaler {
	dc.client = c
	return dc
}

func (dc *Downscaler) Persistence(p *store.Persistence) *Downscaler {
	if p != nil {
		dc.store = p
		dc.persistence = true
	}
	return dc
}

func (dc *Downscaler) Run() (ctrl.Result, error) {
	if err := dc.resetState().createNewClient(); err != nil {
		return ctrl.Result{}, err
	}

	dc.initializeCronTasks()

	return ctrl.Result{}, nil
}

func (dc *Downscaler) addCronJob(scaleStr, namespace string, replicas int) {
	expression := buildCronExpression(dc.recurrence(), scaleStr)
	entryID, err := dc.cron.AddFunc(expression, dc.job(namespace, replicas))
	if err != nil {
		slog.Error("cron", "scheduling error", err)
		return
	}

	dc.log.Info("cron", "assigning new cron entryID", entryID)
}

func (dc *Downscaler) initializeCronTasks() {
	for _, rule := range dc.rules() {
		for _, namespace := range rule.Namespaces {
			dc.addCronJob(rule.UpscaleTime, namespace.String(), upscaleReplicas)
			dc.addCronJob(rule.DownscaleTime, namespace.String(), downscaleReplicas)
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

func (dc *Downscaler) job(namespace string, replicas int) func() {
	return func() {
		for _, rule := range dc.rules() {
			if containsNamespace(namespace, rule.Namespaces) {
				overrideResource := rule.OverrideScaling
				if len(overrideResource) == 0 {
					overrideResource = dc.resourceScaling()
				}

				for _, resource := range overrideResource {
					scaler, err := factory.GetScaler(resource, dc.client, dc.log)
					if err != nil {
						slog.Error("job", "select scaling error", err)
						return
					}

					if err := scaler.Run(namespace, replicas); err != nil {
						slog.Error("job", "resource", resource, "scaling error", err)
					}
				}
			}
		}
	}
}

func containsNamespace(namespace string, namespaces []downscalergov1alpha1.Namespace) bool {
	for _, ns := range namespaces {
		if ns.String() == namespace {
			return true
		}
	}
	return false
}

func (s *Downscaler) Add(ctx context.Context, app downscalergov1alpha1.Downscaler) *Downscaler {
	s.app = app
	return s
}

func (s *Downscaler) Logger(log logr.Logger) *Downscaler {
	s.log = log
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
			return err
		}

		location, err := time.LoadLocation(downscaler.Spec.Schedule.TimeZone)
		if err != nil {
			return err
		}

		cron := cron.New(cron.WithLocation(location))
		dc.cron = cron

		return err
	}

	return nil
}

func buildCronExpression(recurrence, timeStr string) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Error("Invalid time format", "timeStr", timeStr, "error", err)
		return "0 0 * * *"
	}

	if recurrence == "*" || recurrence == "@daily" {
		return fmt.Sprintf("%d %d * * *", t.Minute(), t.Hour())
	}

	return fmt.Sprintf("%d %d * * %s", t.Minute(), t.Hour(), recurrence)
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
