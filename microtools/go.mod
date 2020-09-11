module github.com/hkjojo/go-toolkits/microtools

replace (
	github.com/micro/go-micro/v2 => github.com/hkjojo/go-micro/v2 v2.6.2
	github.com/micro/go-plugins/config/source/consul/v2 => github.com/hkjojo/go-plugins/config/source/consul/v2 v2.5.0
)

go 1.14

require (
	github.com/hashicorp/consul/api v1.4.0
	github.com/hkjojo/go-toolkits/log v1.0.2
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.6.0
	github.com/micro/go-plugins/config/source/consul/v2 v2.5.0
	github.com/sparrc/go-ping v0.0.0-20190613174326-4e5b6552494c
	google.golang.org/grpc v1.26.0
)
