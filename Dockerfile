FROM fireboomapi/base-builder-fireboom:latest as builder
COPY . .
ENV GOPROXY=https://proxy.golang.org,direct
RUN sh scripts/build.sh

FROM fireboomapi/base-runner-fireboom:latest AS final
WORKDIR /app
COPY --from=builder /build/release/fireboom /usr/local/bin/fireboom
#VOLUME [ "store", "upload", "exported", "generated-sdk", "authentication.key", "license.key", "custom-go", "custom-ts", "custom-python", "custom-java" ]
EXPOSE 9123 9991

# ENV FB_API_PUBLIC_URL="http://localhost:9991"
ENV FB_API_INTERNAL_URL="http://localhost:9991"
ENV FB_API_LISTEN_HOST="0.0.0.0"
ENV FB_API_LISTEN_PORT=9991
ENV FB_SERVER_LISTEN_HOST="localhost"
ENV FB_SERVER_LISTEN_PORT=9992
ENV FB_SERVER_URL="http://localhost:9992"
ENV FB_LOG_LEVEL="DEBUG"
ENV FB_REPO_URL_MIRROR="https://git.fireboom.io/{orgName}/{repoName}.git"
ENV FB_RAW_URL_MIRROR="https://raw.git.fireboom.io/{orgName}/{repoName}/{branchName}/{filePath}"

ENTRYPOINT [ "fireboom" ]
CMD [ "start" ]