.PHONY: proto

proto/links/links.pb.go: proto/links/links.proto
	protoc --go_out=. --go_opt=paths=source_relative proto/links/links.proto

proto/token/token.pb.go: proto/token/token.proto
	protoc --go_out=. --go_opt=paths=source_relative proto/token/token.proto

proto: proto/links/links.pb.go proto/token/token.pb.go