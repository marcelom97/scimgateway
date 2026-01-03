module github.com/marcelom97/scimgateway/examples/jwt-auth

go 1.25.5

replace github.com/marcelom97/scimgateway => ../..

require (
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/marcelom97/scimgateway v0.0.0-00010101000000-000000000000
)
