-include CONFIG
-include CREDENTIALS

build:
	go build -o $(APPLICATION)-$(VERSION)/$(APPLICATION) .

create_dir: build
	mkdir -p $(APPLICATION)-$(VERSION)

create_tar: create_dir
	tar -cvzf $(APPLICATION)-$(VERSION).tar.gz $(APPLICATION)-$(VERSION)/$(APPLICATION)

release: create_tar

upload:
	curl -u$(USER):$(PASSWORD) "https://artifactory.verisure.com/artifactory/verisure-generic-snapshots/verisure/$(APPLICATION)/$(VERSION)/$(APPLICATION)-$(VERSION).tar.gz" -T $(APPLICATION)-$(VERSION).tar.gz
