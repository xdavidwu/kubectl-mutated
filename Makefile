.POSIX:

DESTDIR =
PREFIX = /usr/local
BINDIR = $(PREFIX)/bin

GO_SOURCES != find . -name '*.go'

all: kubectl-mutated

kubectl-mutated: $(GO_SOURCES)
	go build -o $@ ./cmd/

install: all
	install -Dm755 kubectl-mutated $(DESTDIR)$(BINDIR)/kubectl-mutated
	ln -s kubectl-mutated $(DESTDIR)$(BINDIR)/kubectl_complete-mutated

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/kubectl-mutated $(DESTDIR)$(BINDIR)/kubectl_complete-mutated

.PHONY: all install uninstall
