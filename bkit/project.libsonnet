local images = import 'images.libsonnet';

local cache = std.native('cache');
local copy = std.native('copy');
local copyFrom = std.native('copyFrom');

local gosources = [
    "go.mod",
    "go.sum",
    "cmd",
    "pkg",
];

local gocache = [
    cache("go-build", "/app/cache"),
    cache("go-mod", "/go/pkg/mod"),
];

{
    project(appIDs):: {
        apiVersion: "bkit/v1",

        targets: {
            all: ["modules", "build", "check"],
        } + {
            modules: ["gotidy", "modulesvendor"],

            gotidy: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                ssh: {},
                command: "go mod tidy",
                output: {
                    artifact: "/app/go.*",
                    "local": ".",
                },
            },

            modulesvendor: {
                from: "gotidy",
                workdir: "/app",
                cache: gocache,
                command: "go mod vendor",
                output: {
                    artifact: "/app/vendor",
                    "local": "vendor",
                },
            },

            build: [appID for appID in appIDs],
        } + {
            [appID]: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                dependsOn: ["modules"],
                command: "go build -trimpath -v -o ./bin/" + appID + " ./cmd/" + appID,
                output: {
                    artifact: "/app/bin/" + appID,
                    "local": "./bin"
                }
            }
            for appID in appIDs
        } + {
            check: ["test", "lint"],

            test: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                command: "go test ./..."
            },

            lint: {
                from: images.golangcilint,
                workdir: "/app",
                cache: gocache,
                copy: [
                    copyFrom("gosources", "/app", "/app"),
                    copy(".golangci.yml", ".golangci.yml"),
                ],
                env: {
                    GOCACHE: "/app/cache/go-build",
                    GOLANGCI_LINT_CACHE: "/app/cache/go-build",
                },
                command: "golangci-lint run"
            },
        } + {
            gobase: {
                from: images.gobuilder,
                workdir: "/app",
                env: {
                    GOCACHE: "/app/cache/go-build",
                    CGO_ENABLED: "0"
                },
                copy: copyFrom('gosources', '/app', '/app')
            },

            gosources: {
                from: "scratch",
                workdir: "/app",
                copy: [copy(source, source) for source in gosources]
            },
        }
    }
}
