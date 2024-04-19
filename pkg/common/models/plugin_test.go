package models

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fmt"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"testing"
)

func TestPluginTree_DeleteByIds(t *testing.T) {
	m := &fileloader.Model[Operation]{
		Root: "../../../store/operation",
		DataRW: &fileloader.MultipleDataRW[Operation]{
			GetDataName: func(item *Operation) string { return item.Path },
			SetDataName: func(item *Operation, name string) { item.Path = name },
			Filter:      func(item *Operation) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Operation) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}
	m.Init()
	dataList := m.ListByCondition(func(item *Operation) bool {
		return item.OperationType == wgpb.OperationType_MUTATION
	})
	path := "CDKEY/RedeemCDKEY"
	insertBytes := []byte(fmt.Sprintf("{\"path\": \"%s\", \"remark\": \"小宝你好呀\", \"cacheConfig\": {\"enabled\": true, \"maxAge\": 111}}", path))
	updateBytes := []byte(fmt.Sprintf("{\"path\": \"%s\", \"enabled\": true, \"cacheConfig\": null}", path))
	insertData, insertErr := m.Insert(insertBytes, "fireboom")
	lockErr := m.TryLockData(path, "fireboom")
	updateData1, updateErr1 := m.UpdateByDataName(updateBytes, "fireboom1")
	getWithLock1, getWithLockErr1 := m.GetWithLockUserByDataName(path)
	updateData2, updateErr2 := m.UpdateByDataName(updateBytes, "fireboom")
	getWithLock2, getWithLockErr2 := m.GetWithLockUserByDataName(path)
	dataTrees, err := m.GetDataTrees()
	fmt.Println(dataList, insertData, insertErr, lockErr, updateData1, updateErr1, updateData2, updateErr2,
		getWithLock1, getWithLock2, getWithLockErr1, getWithLockErr2, dataTrees, err)

}
