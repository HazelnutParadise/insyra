package insyra

func init() {
	SetDefaultConfig()
	initCCLFunctions()
	LogInfo("", "", "Thank you for using Insyra %s(v%s)!\nOfficial website: https://insyra.hazelnut-paradise.com\n\n", VersionName, Version)
}
