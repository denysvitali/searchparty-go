PROJECT_NAME=denysvitali/searchparty-go

bufbuild:
	TEMP_DIR=$$(mktemp -d); \
		docker build -t "$(PROJECT_NAME)-bufbuild" -f Dockerfile.buf "$$TEMP_DIR"; \
		rm -rf "$$TEMP_DIR"

genproto: bufbuild
	docker run \
		-u "$$(id -u):$$(id -g)" \
		--rm \
		-v "$$PWD:/app" \
		-e "HOME=/tmp" \
		-w /app \
		"$(PROJECT_NAME)-bufbuild" generate
