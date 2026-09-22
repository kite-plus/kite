# A self-contained Kite server.
#
# The binary is compiled here rather than copied out of a release archive, so
# that `docker build .` works from any checkout and a contributor can try a
# change in a container without publishing anything first. The versions below
# are pinned for the same reason they are pinned everywhere else in this
# repository: the same commit should produce the same image next year.

# Must match the toolchain directive in go.mod.
ARG GO_VERSION=1.26.8
# Must match web/.nvmrc.
ARG NODE_VERSION=22.19.0
ARG ALPINE_VERSION=3.22

# The admin is embedded in the binary, so it has to exist before the binary is
# built. Its own stage keeps Node out of the image that ships.
FROM node:${NODE_VERSION}-alpine AS admin
WORKDIR /src/web

# The manifest and the lockfile alone first: dependencies change far less
# often than the code does, and this layer is then reused across builds.
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=admin /src/web/dist ./web/dist

# Stamped rather than read from the repository, because .git is not in the
# build context: an image should know what it is without carrying history.
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

# No cgo, for the same reason the released binaries have none: one
# self-contained executable, and a dependency that quietly needs a C
# toolchain fails here rather than at run time.
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags "-s -w \
      -X github.com/kite-plus/kite/internal/buildinfo.Version=${VERSION} \
      -X github.com/kite-plus/kite/internal/buildinfo.Commit=${COMMIT} \
      -X github.com/kite-plus/kite/internal/buildinfo.Date=${DATE}" \
    -o /out/kite ./cmd/kite

FROM alpine:${ALPINE_VERSION}

# git is not optional: publishing commits the content, and a container that
# cannot do that is a Kite with half its job missing. ca-certificates is for
# pushing over https, tzdata so that dates render in the site's own zone.
RUN apk add --no-cache ca-certificates git tzdata && \
    adduser -D -u 1717 -h /data kite

COPY --from=build /out/kite /usr/local/bin/kite
COPY docker/entrypoint.sh /usr/local/bin/kite-entrypoint

# The project lives in a volume, not in the image. An image holding a site
# would have to be rebuilt to publish a post.
WORKDIR /data
VOLUME /data
USER kite
EXPOSE 1717

# The site itself, not the API: it answers whether or not the studio is
# mounted, and whether or not this server has been set up yet.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:1717/ || exit 1

ENTRYPOINT ["kite-entrypoint"]
CMD ["serve"]
