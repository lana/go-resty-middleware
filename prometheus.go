package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	prom "github.com/prometheus/client_golang/prometheus"
)

const defaultSubsystem = "resty"

type ctxkey struct{ name string }

var urlKey = &ctxkey{"request-url"}

type prometheus struct {
	reqTotal *prom.CounterVec
	reqDur   *prom.HistogramVec
}

func (p *prometheus) register(reg prom.Registerer, subsystem string) {
	if subsystem == "" {
		subsystem = defaultSubsystem
	}

	p.reqTotal = prom.NewCounterVec(prom.CounterOpts{
		Name:      "requests_total",
		Subsystem: subsystem,
		Help:      "The number of requests made",
	}, []string{"code", "method", "host", "url"})

	p.reqDur = prom.NewHistogramVec(prom.HistogramOpts{
		Name:      "request_duration_seconds",
		Subsystem: subsystem,
		Help:      "The request latency in seconds",
	}, []string{"code", "method", "host", "url"})

	reg.MustRegister(p.reqTotal)
	reg.MustRegister(p.reqDur)
}

func (p *prometheus) collect(req *http.Request, code int, dur time.Duration) {
	url, _ := req.Context().Value(urlKey).(string)

	values := []string{
		strconv.Itoa(code),
		req.Method,
		req.URL.Hostname(),
		url,
	}

	p.reqTotal.WithLabelValues(values...).Inc()
	p.reqDur.WithLabelValues(values...).Observe(dur.Seconds())
}

func (p *prometheus) beforeRequest(client *resty.Client, req *resty.Request) error {
	ctx := context.WithValue(req.Context(), urlKey, req.URL)
	req.SetContext(ctx)

	return nil
}

func (p *prometheus) collectAfterResponse(client *resty.Client, res *resty.Response) error {
	p.collect(
		res.Request.RawRequest,
		res.StatusCode(),
		res.Time(),
	)

	return nil
}

func (p *prometheus) collectError(req *resty.Request, err error) {
	code := http.StatusInternalServerError

	var dur time.Duration
	var e *resty.ResponseError

	if errors.As(err, &e) {
		code = e.Response.StatusCode()
		dur = e.Response.Time()
	}

	p.collect(
		req.RawRequest,
		code,
		dur,
	)
}

// Prometheus generates a new set of metrics with a certain subsystem name from
// resty requests.
func Prometheus(client *resty.Client, subsystem string) *resty.Client {
	return PrometheusWithRegister(client, prom.DefaultRegisterer, subsystem)
}

// PrometheusWithRegister generates a new set of metrics with a certain
// subsystem name from resty requests with a custom prometheus registerer.
func PrometheusWithRegister(client *resty.Client, reg prom.Registerer, subsystem string) *resty.Client {
	p := &prometheus{}
	p.register(reg, subsystem)

	return client.
		OnBeforeRequest(p.beforeRequest).
		OnAfterResponse(p.collectAfterResponse).
		OnError(p.collectError)
}
