package cloudscheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type JobsMetric struct {
	CountSubmitted int
	CountRunning   int
	CountCompleted int
}

func (m *JobsMetric) GrantTotal() int {
	return m.CountSubmitted + m.CountRunning + m.CountCompleted
}

type MetricsCollector struct {
	cs             *CloudScheduler
	jobsGrandTotal *prometheus.Desc
	jobsTotal      *prometheus.Desc
}

func NewMetricsCollector(cs *CloudScheduler) *MetricsCollector {
	return &MetricsCollector{
		cs: cs,
		jobsGrandTotal: prometheus.NewDesc(
			"scheduler_jobs_total",
			"Total number of jobs in the scheduler",
			nil,
			nil),
		jobsTotal: prometheus.NewDesc(
			"scheduler_jobs_count",
			"Number of jobs per status",
			[]string{"status"},
			nil),
	}
}

func (mc *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- mc.jobsGrandTotal
	ch <- mc.jobsTotal
}

func (mc *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	var m JobsMetric
	// TODO: Do we count removed jobs for the completed jobs?
	for _, j := range mc.cs.GoalManager.GetJobs() {
		switch j.Status {
		case datatype.JobCreated, datatype.JobDrafted, datatype.JobSuspended, datatype.JobSubmitted:
			m.CountSubmitted += 1
		case datatype.JobRunning:
			m.CountRunning += 1
		case datatype.JobComplete, datatype.JobRemoved:
			m.CountCompleted += 1
		}
	}
	ch <- prometheus.MustNewConstMetric(
		mc.jobsGrandTotal,
		prometheus.GaugeValue,
		float64(m.GrantTotal()),
	)
	ch <- prometheus.MustNewConstMetric(
		mc.jobsTotal,
		prometheus.GaugeValue,
		float64(m.CountSubmitted),
		"submitted",
	)
	ch <- prometheus.MustNewConstMetric(
		mc.jobsTotal,
		prometheus.GaugeValue,
		float64(m.CountRunning),
		"running",
	)
	ch <- prometheus.MustNewConstMetric(
		mc.jobsTotal,
		prometheus.GaugeValue,
		float64(m.CountCompleted),
		"completed",
	)
}
