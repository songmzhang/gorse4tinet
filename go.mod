module github.com/zhenghaoz/gorse

go 1.14

require (
	github.com/ReneKroon/ttlcache/v2 v2.7.0
	github.com/alicebob/miniredis/v2 v2.16.1
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/bits-and-blooms/bitset v1.2.1
	github.com/chewxy/math32 v1.0.8
	github.com/emicklei/go-restful-openapi/v2 v2.3.0
	github.com/emicklei/go-restful/v3 v3.5.2
	github.com/go-redis/redis/v8 v8.11.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/securecookie v1.1.1
	github.com/gorse-io/dashboard v0.0.0-20220222014325-2ec7a544dff4
	github.com/haxii/go-swagger-ui v3.19.4+incompatible
	github.com/json-iterator/go v1.1.12
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20210324180055-18c50b0c2098 // indirect
	github.com/klauspost/cpuid/v2 v2.0.10
	github.com/lafikl/consistent v0.0.0-20210222184039-5e8acd7e59f2
	github.com/lib/pq v1.10.2
	github.com/mailru/go-clickhouse v1.6.0
	github.com/orcaman/concurrent-map v1.0.0
	github.com/prometheus/client_golang v1.10.0
	github.com/rakyll/statik v0.1.7
	github.com/scylladb/go-set v1.0.2
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.10.1
	github.com/steinfletcher/apitest v1.5.11
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.9.1
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	go.mongodb.org/mongo-driver v1.5.1
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.17.0
	gonum.org/v1/gonum v0.0.0-20190409070159-6e46824336d2
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	modernc.org/mathutil v1.4.1
	modernc.org/sortutil v1.1.0
)

replace github.com/gorse-io/dashboard => github.com/oufangwei/dashboard v0.0.0-20220222014325-2ec7a544dff4