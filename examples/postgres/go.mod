module github.com/marcelom97/scimgateway/examples/postgres

go 1.25

replace github.com/marcelom97/scimgateway => ../..

require (
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/marcelom97/scimgateway v0.0.0-00010101000000-000000000000
)
