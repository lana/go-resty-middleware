package middleware

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"gitlab.com/flimzy/testy"
)

const testHost = "http://flock.service"

func TestPrometheus(t *testing.T) {
	type tt struct {
		url    string
		params map[string]string
		err    string
	}

	tests := testy.NewTable()

	tests.Add("after response", func() interface{} {
		httpmock.RegisterResponder(http.MethodGet, testHost+"/1",
			httpmock.NewStringResponder(http.StatusOK, ""))

		return tt{
			url: "/1",
		}
	})

	tests.Add("error", func() interface{} {
		httpmock.RegisterResponder(http.MethodGet, testHost+"/1",
			httpmock.NewErrorResponder(errors.New("failed")))

		return tt{
			url: "/1",
			err: `Get "http://flock.service/1": failed`,
		}
	})

	tests.Add("path params", func() interface{} {
		httpmock.RegisterResponder(http.MethodGet, testHost+"/10",
			httpmock.NewStringResponder(http.StatusOK, ""))

		return tt{
			url: "/{id}",
			params: map[string]string{
				"id": "10",
			},
		}
	})

	tests.Run(t, func(t *testing.T, tt tt) {
		reg := prom.NewRegistry()
		client := PrometheusWithRegister(resty.New(), reg, defaultSubsystem)

		httpmock.ActivateNonDefault(client.GetClient())
		defer httpmock.DeactivateAndReset()

		_, err := client.SetHostURL(testHost).R().SetPathParams(tt.params).Get(tt.url)

		if total, _ := testutil.GatherAndCount(reg, "resty_requests_total"); total < 1 {
			t.Errorf("expected at least 1, got %d", total)
		}

		if err := testutil.GatherAndCompare(reg, testy.Snapshot(t), "resty_requests_total"); err != nil {
			t.Error(err)
		}

		if total, _ := testutil.GatherAndCount(reg, "resty_request_duration_seconds"); total < 1 {
			t.Errorf("expected at least 1, got %d", total)
		}

		testy.Error(t, tt.err, err)
	})

	t.Run("panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected a panic, got nil")
			}
		}()

		Prometheus(resty.New(), "")
		Prometheus(resty.New(), "")
	})
}
