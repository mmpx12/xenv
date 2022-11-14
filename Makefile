build:
	go build -ldflags="-w -s" 

install:
	mv xenv /usr/bin/xenv

termux-install:
	mv xenv /data/data/com.termux/files/usr/bin/xenv

all: build install

termux-all: build termux-install

clean:
	rm -f xenv /usr/bin/xenv

termux-clean:
	rm -f xenv /data/data/com.termux/files/usr/bin/xenv
