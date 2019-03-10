
#export GOOS=linux
#export GOARCH=386

.PHONY: sync fmt vet

all: vet fmt MovieNight MovieNight.exe

MovieNight.exe: *.go
	GOOS=windows GOARCH=amd64 go build -o MovieNight.exe

MovieNight: *.go
	GOOS=linux GOARCH=386 go build -o MovieNight

clean:
	rm MovieNight.exe MovieNight

fmt:
	gofmt -w .

vet:
	go vet

sync:
	#rsync -v --no-perms --chmod=ugo=rwX -r ./ zorchenhimer@movienight.zorchenhimer.com:/home/zorchenhimer/movienight/
	#rsync -v --no-perms --chmod=ugo=rwX -e "ssh -i /c/movienight/movienight-deploy.key" -r ./ zorchenhimer@movienight.zorchenhimer.com:/home/zorchenhimer/movienight/
	scp -i /c/movienight/movienight-deploy.key -r . zorchenhimer@movienight.zorchenhimer.com:/home/zorchenhimer/movienight
