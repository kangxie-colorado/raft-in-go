module github.com/kangxie-colorado/golang-primer/messaging/test/lib

go 1.16

replace github.com/kangxie-colorado/golang-primer/messaging/lib => ../../lib

require (
	github.com/kangxie-colorado/golang-primer/messaging/lib v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
)
