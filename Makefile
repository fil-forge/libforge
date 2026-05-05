.PHONY: gen clean-gen

gen:
	go generate ./...

clean-gen:
	find capabilities -path '*/datamodel/cbor_gen*.go' -delete
	find capabilities -path '*/datamodel/json_gen*.go' -delete
