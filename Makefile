gooseUp:
	goose sqlite3 -dir ./sql/schema/ ./sql/vb.db up

gooseDown:
	goose sqlite3 -dir ./sql/schema/ ./sql/vb.db down

seed:
	goose sqlite3 -dir ./sql/seed/ ./sql/vb.db -no-versioning up

sqlc:
	cd ./sql/queries/ && sqlc generate

test:
	go test ./...

run:
	go build && ./vl_hl

clean:
	go clean

