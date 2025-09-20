package insyra

func init() {
	SetDefaultConfig()
	initCCLFunctions()
	LogInfo("", "", "Thank you for adopting Insyra %s(v%s)!\nOfficial website: https://insyra.hazelnut-paradise.com\n\n", VersionName, Version)
}
