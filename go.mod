module github.com/hopeio/scaffold

go 1.23.0

require (
	github.com/danielvladco/go-proto-gql v0.10.1-0.20221227181908-22fca0a9469c
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0
	github.com/hopeio/cherry v1.0.0
	github.com/hopeio/context v0.1.2
	github.com/hopeio/initialize v1.0.0
	github.com/hopeio/protobuf v1.0.0
	github.com/hopeio/utils v1.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20241209162323-e6fa225c2576
	google.golang.org/grpc v1.69.0
	google.golang.org/protobuf v1.35.2
)
replace (
	github.com/hopeio/cherry => ../cherry
	github.com/hopeio/context => ../context
	github.com/hopeio/initialize => ../initialize
	github.com/hopeio/protobuf => ../protobuf
	github.com/hopeio/utils => ../utils
)
