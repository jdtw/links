.PHONY: proto

proto/links/links.pb.go: proto/links/links.proto
	protoc --go_out=. --go_opt=paths=source_relative proto/links/links.proto

proto: proto/links/links.pb.go