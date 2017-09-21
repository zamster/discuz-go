# Discuz golang makefile

all: main

install: all
	cp discuz /usr/local/bin/
	cp discuzd /usr/local/bin/
	touch /var/log/discuz.log
	touch /var/run/discuz.pid

monit:
	cp discuz.monit /etc/monit/conf.d/discuz

main: discuz.6 main.6
	6l -o discuzd main.6
	rm -rf *.6

discuz.6: discuz.go cache.go db.go cookie.go
	6g discuz.go cache.go db.go cookie.go

main.6: main.go
	6g main.go

clean:
	rm discuzd
	rm -rf *.6
	rm -rf /usr/local/bin/discuz
	rm -rf /usr/local/bin/discuzd

fmt:
	gofmt -spaces=true -tabwidth=4 -w=true -tabindent=false ./