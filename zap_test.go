package middleware

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"gitlab.com/flimzy/testy"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestZap(t *testing.T) {
	type tt struct {
		url      string
		err      string
		logCount int
		logMsg   string
	}

	tests := testy.NewTable()

	tests.Add("after response", func() interface{} {
		httpmock.RegisterResponder(http.MethodGet, testHost+"/1",
			httpmock.NewStringResponder(http.StatusOK, "OK"))

		return tt{
			url:      "/1",
			logCount: 1,
			logMsg:   "request ok",
		}
	})

	tests.Add("error", func() interface{} {
		httpmock.RegisterResponder(http.MethodGet, testHost+"/1",
			httpmock.NewErrorResponder(errors.New("failed")))

		return tt{
			url:      "/1",
			err:      `Get "http://flock.service/1": failed`,
			logCount: 1,
			logMsg:   "request error",
		}
	})

	tests.Run(t, func(t *testing.T, tt tt) {
		observedZapCore, observedLogs := observer.New(zap.DebugLevel)
		logger := zap.New(observedZapCore)
		defer logger.Sync()

		client := Zap(resty.New(), logger)

		httpmock.ActivateNonDefault(client.GetClient())
		defer httpmock.DeactivateAndReset()

		_, err := client.SetHostURL(testHost).R().Get(tt.url)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		} else {
			assert.NoError(t, err)
		}

		logs := observedLogs.All()
		assert.Equal(t, tt.logCount, len(logs))
		assert.Equal(t, tt.logMsg, logs[0].Message)
	})
}
