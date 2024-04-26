%.pb.go: %.proto
	protoc --go_opt=paths=source_relative --go_out=. --proto_path=. $<

.PHONY: clean
clean:
	-rm *.pb.go
