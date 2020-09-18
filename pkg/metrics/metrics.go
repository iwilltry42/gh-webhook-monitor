package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CodeGroup struct {
	Name       string
	LowerBound int
	UpperBound int
}

var (
	CodeGroupUnused = CodeGroup{
		Name:       "unused",
		LowerBound: 0,
		UpperBound: 0,
	}
	CodeGroup2xx = CodeGroup{
		Name:       "2xx",
		LowerBound: 200,
		UpperBound: 299,
	}
	CodeGroup3xx = CodeGroup{
		Name:       "3xx",
		LowerBound: 300,
		UpperBound: 399,
	}
	CodeGroup4xx = CodeGroup{
		Name:       "4xx",
		LowerBound: 400,
		UpperBound: 499,
	}
	CodeGroup5xx = CodeGroup{
		Name:       "5xx",
		LowerBound: 500,
		UpperBound: 599,
	}
	CodeGroupOthers = CodeGroup{
		Name:       "xxx",
		LowerBound: 999,
		UpperBound: 999,
	}

	CodeGroups = []CodeGroup{
		CodeGroupUnused,
		CodeGroup2xx,
		CodeGroup3xx,
		CodeGroup4xx,
		CodeGroup5xx,
	}

	WebhookLastStatusCodeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gh_webhook_last_status_code_total",
		Help: "Total Number of Status Codes collected from the 'Last Webhook Response'",
	}, []string{
		"repository",
		"webhook",
		"target",
		"status",
		"code",
		"code_group",
	})

	WebhookLastStatusCodeGroup = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gh_webhook_last_status_code_group",
		Help: "The last HTTP status code per webhook (1 = active)",
	}, []string{
		"repository",
		"webhook",
		"target",
		"status",
		"code_group",
	})

	RepositoryFailedWebhookListTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gh_webhooks_repository_list_failed_total",
		Help: "Total number of failed webhook lists per repository",
	}, []string{
		"repository",
		"error",
	})
)
