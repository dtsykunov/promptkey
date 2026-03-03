FRONTEND := promptkey/frontend
GODIR    := ./promptkey
OUTDIR   := build/windows

.PHONY: frontend build-windows build-windows-debug clean

$(OUTDIR):
	mkdir -p $(OUTDIR)

frontend:
	cd $(FRONTEND) && npm ci && npm run build

# Release: GUI subsystem — no console window, for distribution
build-windows: frontend | $(OUTDIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -ldflags "-H windowsgui" \
	-o $(OUTDIR)/promptkey.exe \
	$(GODIR)

# Debug: console subsystem — terminal stays open, logs are visible
build-windows-debug: frontend | $(OUTDIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build \
	-o $(OUTDIR)/promptkey-debug.exe \
	$(GODIR)

clean:
	rm -rf build/ $(FRONTEND)/dist
