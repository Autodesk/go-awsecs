module github.com/Autodesk/go-awsecs

go 1.13

require (
	github.com/aws/aws-sdk-go v1.20.21
	github.com/cenkalti/backoff v0.0.0-00010101000000-000000000000
	github.com/sergi/go-diff v1.0.0
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/net v0.0.0-20200219183655-46282727080f // indirect
)

replace github.com/cenkalti/backoff => github.com/cenkalti/backoff/v3 v3.1.1
