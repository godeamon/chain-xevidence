build: 
	rm -rf output
	mkdir output
	go build -o output/xevidence cmd/xevidence/*.go 
	cp -r conf output