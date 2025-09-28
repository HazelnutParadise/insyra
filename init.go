// `insyra` main package provides unified interfaces and structures for data manipulation and analysis.
package insyra

func init() {
	SetDefaultConfig()
	initCCLFunctions()
	go LogInfo("", "", "Welcome to Insyra %s(v%s)!!\nOfficial website: https://insyra.hazelnut-paradise.com\n\n", VersionName, Version)
}
