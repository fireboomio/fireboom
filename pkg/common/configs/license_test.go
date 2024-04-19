package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fmt"
	"testing"
	"time"
)

func TestGenerateCommunityCode(t *testing.T) {
	generateUserLicense(consts.LicenseTypeCommunity, 12, map[string]int{
		consts.LicenseOperation:  888,
		consts.LicenseDatasource: 18,
	})
}

func TestGenerateEnterpriseCode(t *testing.T) {
	generateUserLicense(consts.LicenseTypeEnterprise, 12, map[string]int{
		consts.LicenseOperation:        -1,
		consts.LicenseDatasource:       -1,
		consts.LicenseImport:           1,
		consts.LicenseTeamwork:         1,
		consts.LicensePrismaDatasource: 1,
		consts.LicenseIncrementBuild:   1,
	})
}

func generateUserLicense(licenseType consts.LicenseType, months int, userLimits map[string]int) {
	requiredUserCode := "d70f6e1d2083fd88739f81ba87030339"
	encodeCode := utils.GenerateLicenseKey(requiredUserCode, &userLicense{
		Type:       licenseType,
		ExpireTime: time.Now().AddDate(0, months, 0),
		UserLimits: userLimits,
	})
	fmt.Println(encodeCode)
}
