// Package consts
/*
 license常量
*/
package consts

const LicenseStatusField = "licenseStatus"

// license status
const (
	LicenseStatusEmpty   = "empty"
	LicenseStatusInvalid = "invalid"
	LicenseStatusExpired = "expired"
	LicenseStatusLimited = "limited"
)

type LicenseType string

// license type
const (
	LicenseTypeCommunity    LicenseType = "community"
	LicenseTypeProfessional LicenseType = "professional"
	LicenseTypeEnterprise   LicenseType = "enterprise"
)

// license module
const (
	LicenseOperation        = "operation"
	LicenseDatasource       = "datasource"
	LicenseImport           = "import"
	LicenseTeamwork         = "teamwork"
	LicensePrismaDatasource = "prismaDatasource"
	LicenseIncrementBuild   = EngineIncrementBuild
)
