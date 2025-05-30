package insyra

func init() {
	SetDefaultConfig()
	initCCLFunctions()
	LogInfo("Thank you for using Insyra v%s!\nOfficial website: https://insyra.hazelnut-paradise.com\n\n", Version)
}
