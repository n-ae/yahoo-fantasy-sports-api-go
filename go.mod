module github.com/n-ae/yahoo-fantasy-sports-api-go

go 1.25.0

retract (
	v1.4.9-extension.1 // mispublished
	v1.4.9 // mispublished
	v0.2.2 // abandoned v0.x line; shares a commit with v1.4.10, do not use
	v0.2.1 // abandoned v0.x line, do not use
)

require github.com/mattn/go-sqlite3 v1.14.32
