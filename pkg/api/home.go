// Package api
/*
 注册首页统计的路由
 统计数据源，接口，认证，存储等信息
*/
package api

import (
	"fireboom-server/pkg/common/models"
	"github.com/labstack/echo/v4"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"net/http"
)

func HomeRouter(contextRouter *echo.Group) {
	handler := &home{}
	homeRouter := contextRouter.Group("/home")
	homeRouter.GET("", handler.getHomeData)
	homeRouter.GET("/bulletin", handler.getBulletin)
}

type home struct{}

type (
	homeStatistics struct {
		DataSource     *datasourceStatistics     `json:"dataSource"`
		Operation      *operationStatistics      `json:"operation"`
		Authentication *authenticationStatistics `json:"authentication"`
		Storage        *storageStatistics        `json:"storage"`
	}
	datasourceStatistics struct {
		RestTotal      int `json:"restTotal"`
		GraphqlTotal   int `json:"graphqlTotal"`
		DatabaseTotal  int `json:"databaseTotal"`
		CustomizeTotal int `json:"customizeTotal"`
	}
	operationStatistics struct {
		QueryTotal        int `json:"queryTotal"`
		MutationTotal     int `json:"mutationTotal"`
		SubscriptionTotal int `json:"subscriptionTotal"`
		LiveQueryTotal    int `json:"liveQueryTotal"`
	}
	authenticationStatistics struct {
		AuthenticationTotal int `json:"authenticationTotal"`
		UserTotal           int `json:"userTotal"`
		UserTodayAdd        int `json:"userTodayAdd"`
	}

	storageStatistics struct {
		StorageTotal int `json:"storageTotal"`
		MemoryTotal  int `json:"memoryTotal"`
		MemoryUsed   int `json:"memoryUsed"`
	}
)

// @Tags home
// @Description "首页统计数据"
// @Success 200 {object} homeStatistics "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /home [get]
func (h *home) getHomeData(c echo.Context) error {
	countDatasourceFunc := func(filterFuc func(*models.Datasource) bool) int {
		filter := func(item *models.Datasource) bool {
			return item.Enabled && filterFuc(item)
		}
		return len(models.DatasourceRoot.ListByCondition(filter))
	}
	homeDatasource := &datasourceStatistics{
		DatabaseTotal: countDatasourceFunc(func(item *models.Datasource) bool { return item.IsCustomDatabase() }),
		RestTotal:     countDatasourceFunc(func(item *models.Datasource) bool { return item.Kind == wgpb.DataSourceKind_REST }),
		GraphqlTotal:  countDatasourceFunc(func(item *models.Datasource) bool { return item.Kind == wgpb.DataSourceKind_GRAPHQL }),
	}

	countOperationFunc := func(filterFuc func(*models.Operation) bool) int {
		filter := func(item *models.Operation) bool {
			return item.Enabled && filterFuc(item)
		}
		return len(models.OperationRoot.ListByCondition(filter))
	}
	homeOperation := &operationStatistics{
		QueryTotal:        countOperationFunc(func(item *models.Operation) bool { return item.OperationType == wgpb.OperationType_QUERY }),
		MutationTotal:     countOperationFunc(func(item *models.Operation) bool { return item.OperationType == wgpb.OperationType_MUTATION }),
		SubscriptionTotal: countOperationFunc(func(item *models.Operation) bool { return item.OperationType == wgpb.OperationType_SUBSCRIPTION }),
		LiveQueryTotal:    countOperationFunc(func(item *models.Operation) bool { return item.LiveQueryConfig != nil && item.LiveQueryConfig.Enabled }),
	}

	homeAuthentication := &authenticationStatistics{
		AuthenticationTotal: len(models.AuthenticationRoot.ListByCondition(func(item *models.Authentication) bool { return item.Enabled })),
	}

	homeStorage := &storageStatistics{
		StorageTotal: len(models.StorageRoot.ListByCondition(func(item *models.Storage) bool { return item.Enabled })),
	}

	return c.JSON(http.StatusOK, &homeStatistics{
		DataSource:     homeDatasource,
		Operation:      homeOperation,
		Authentication: homeAuthentication,
		Storage:        homeStorage,
	})
}

// TODO: system notify
func (h *home) getBulletin(c echo.Context) error {
	return c.JSON(http.StatusOK, []map[string]any{{
		"bulletinType": 1,
		"title":        "震惊!!!!",
		"date":         "30分钟前",
	}})
}
