gooseUp:
	goose sqlite3 -dir ./sql/schema/ ./sql/vlhl.db up

gooseDown:
	goose sqlite3 -dir ./sql/schema/ ./sql/vlhl.db down

seed:
	goose sqlite3 -dir ./sql/seed/ ./sql/vlhl.db -no-versioning up

sqlc:
	cd ./sql/queries/ && sqlc generate

test:
	go test ./...

run:
	go build && ./vl_hl

clean:
	go clean

