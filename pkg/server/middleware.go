// Package server
/*
 自定义echo中间件
*/
package server

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fmt"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/exp/slices"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ProductionAuthentication 生产环境下的9123端口鉴权
func ProductionAuthentication(productionAuthenticationKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			urlPath := c.Request().URL.Path
			authenticationUrls := configs.ApplicationData.AuthenticationUrls
			if !slices.ContainsFunc(authenticationUrls, func(item string) bool { return strings.HasPrefix(urlPath, item) }) {
				return next(c)
			}

			requestAuthenticationKey := c.Request().URL.Query().Get(consts.QueryParamAuthentication)
			if requestAuthenticationKey == "" {
				requestAuthenticationKey = c.Request().Header.Get(consts.HeaderParamAuthentication)
			}

			if requestAuthenticationKey == productionAuthenticationKey {
				return next(c)
			}

			return echo.NewHTTPError(http.StatusUnauthorized)
		}
	}
}

// CORS will handle the CORS middleware
func CORS(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		c.Response().Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
		c.Response().Header().Add("Access-Control-Allow-Credentials", "true")                                                    //设置为true，允许ajax异步请求带cookie信息
		c.Response().Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                             //允许请求方法
		return next(c)
	}
}

func RequestLoggerWithConfig() echo.MiddlewareFunc {
	return echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		LogURI:      true,
		LogRemoteIP: true,
		LogMethod:   true,
		LogURIPath:  true,
		LogStatus:   true,
		LogLatency:  true,
		LogError:    true,
		Skipper: func(c echo.Context) bool {
			urlPath := c.Request().URL.Path
			requestLoggerSkippers := configs.ApplicationData.RequestLoggerSkippers
			return slices.ContainsFunc(requestLoggerSkippers, func(item string) bool {
				return strings.Contains(urlPath, item)
			})
		},
		BeforeNextFunc: func(c echo.Context) {
			c.Set("customValueFromContext", 42)
		},
		LogValuesFunc: func(c echo.Context, v echoMiddleware.RequestLoggerValues) error {
			logger.Info(fmt.Sprintf("%s %s %d %v %v", v.Method, v.URI, v.Status, v.Latency, v.Error))
			return nil
		},
	})
}

// RequestID will handle the LOGS middleware
func RequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Request().Header.Set(echo.HeaderXRequestID, uuid.NewString())
		return next(c)
	}
}

// GzipWithConfig will handle the LOGS middleware
func GzipWithConfig() echo.MiddlewareFunc {
	return echoMiddleware.GzipWithConfig(echoMiddleware.GzipConfig{Level: 5})
}
