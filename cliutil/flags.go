package cliutil

import "flag"

// SetupServiceFlagSet 统一设置服务相关命令的 flagSet
func SetupServiceFlagSet(flagSet *flag.FlagSet, serviceName *string) {
	flagSet.StringVar(serviceName, "name", "telegram-faq-bot", "服务名称")
	flagSet.StringVar(serviceName, "n", "telegram-faq-bot", "服务名称 (简写)")
}
