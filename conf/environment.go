package conf

// environment
type EnvironmentEnum int8

const (
	ExampleEnvironmentEnum EnvironmentEnum = 0x01
	MainnetEnvironmentEnum EnvironmentEnum = 0x02
	TestnetEnvironmentEnum EnvironmentEnum = 0x03
	LocalEnvironmentEnum   EnvironmentEnum = 0x04
)

var SystemEnvironmentEnum = LocalEnvironmentEnum

func GetYaml() string {
	var (
		//ConfigFile = "conf/conf_example.yaml"
		ConfigFile = "conf/conf_pro.yaml"
	)
	if SystemEnvironmentEnum == MainnetEnvironmentEnum {
		ConfigFile = "conf/conf_pro.yaml"
	} else if SystemEnvironmentEnum == ExampleEnvironmentEnum {
		ConfigFile = "conf/conf_example.yaml"
	} else if SystemEnvironmentEnum == TestnetEnvironmentEnum {
		ConfigFile = "conf/conf_test.yaml"
	} else if SystemEnvironmentEnum == LocalEnvironmentEnum {
		ConfigFile = "conf/conf_loc.yaml"
	}
	return ConfigFile
}
