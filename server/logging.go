package server

import (
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is a middleware and zap to provide an "access log" like logging for each request.
func ZapLogger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			req := c.Request()

			requestID := req.Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.Must(uuid.NewV7()).String()
				req.Header.Set(echo.HeaderXRequestID, requestID)
			}

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			stop := time.Now()
			res := c.Response()
			res.Header().Set(echo.HeaderXRequestID, requestID)

			contentLength := req.Header.Get(echo.HeaderContentLength)
			if contentLength == "" {
				contentLength = "0"
			}

			latency := stop.Sub(start)

			const preferredBase = 10
			fields := []zapcore.Field{
				zap.String("bytes_in", contentLength),
				zap.String("bytes_out", strconv.FormatInt(res.Size, preferredBase)),
				zap.String("latency", strconv.FormatInt(int64(latency), preferredBase)),
				zap.Int("status", res.Status),
				zap.String("host", req.Host),
				zap.String("id", requestID),
				zap.String("latency_human", stop.Sub(start).String()),
				zap.String("method", req.Method),
				zap.String("remote_ip", c.RealIP()),
				zap.String("time", time.Now().Format(time.RFC3339Nano)),
				zap.String("uri", req.RequestURI),
				zap.String("user_agent", req.UserAgent()),
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			log.Info("http request", fields...)

			return nil
		}
	}
}
