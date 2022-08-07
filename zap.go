package middleware

import (
	"errors"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type zapLogger struct {
	logger *zap.Logger
}

func (z *zapLogger) logAfterResponse(client *resty.Client, res *resty.Response) error {
	if res.IsError() {
		z.log(
			z.logger.Error, "request error", res.Request, res, zap.Any("error", res.Error()),
		)
	} else {
		if z.logger.Core().Enabled(zap.DebugLevel) {
			z.log(z.logger.Debug, "request ok", res.Request, res)
		}
	}

	return nil
}

func (z *zapLogger) logError(req *resty.Request, err error) {
	var res *resty.Response
	var e *resty.ResponseError

	if errors.As(err, &e) {
		res = e.Response
	}

	z.log(z.logger.Error, "request error", req, res, zap.Error(err))
}

func (z *zapLogger) log(
	log func(msg string, fields ...zap.Field),
	msg string,
	req *resty.Request,
	res *resty.Response,
	fields ...zap.Field,
) {
	url := req.RawRequest.URL

	fields = append(
		fields,
		zap.String("host", url.Hostname()),
		zap.String("url", url.String()),
		zap.String("method", req.Method),
	)

	if req.Body != nil {
		fields = append(fields, zap.Any("reqBody", req.Body))
	} else if len(req.FormData) > 0 {
		fields = append(fields, zap.Any("reqBody", req.FormData))
	}

	var code int
	var duration time.Duration
	var resBody string

	if res != nil {
		code = res.StatusCode()
		duration = res.Time()
		resBody = res.String()
	}

	fields = append(
		fields,
		zap.Int("code", code),
		zap.Duration("duration", duration),
		zap.String("resBody", resBody),
	)

	log(msg, fields...)
}

// Zap logs the request and response from resty requests.
func Zap(client *resty.Client, logger *zap.Logger) *resty.Client {
	z := &zapLogger{logger: logger}

	return client.SetLogger(logger.Sugar()).
		OnAfterResponse(z.logAfterResponse).
		OnError(z.logError)
}
