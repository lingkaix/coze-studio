gotestsum --hide-summary=skipped,passed --format=standard-quiet -- ./... -gcflags=all=-N -gcflags=all=-l

go test ./api/handler/coze -run TestAggregateStreamVariables -v -count=1 -gcflags=all=-N -gcflags=all=-l