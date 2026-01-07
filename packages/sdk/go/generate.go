package sdk

//go:generate go run ./cmd/normalize_openapi -in ../../agent/docs/swagger.yaml -out ./openapi/swagger.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml ./openapi/swagger.yaml
