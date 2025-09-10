# Go Project Template

This repository provides example code for setting up an empty Go project following best practices.
For more information on recommended project structure, please look at this info [Golang Standards Project Layout](https://github.com/golang-standards/project-layout).

# External Packages
The template project uses the following external packages:
- A CLI based on the [Cobra](https://github.com/spf13/cobra) framework. This framework is used by many other Golang project, e.g Kubernetes, Docker etc.
- Logging via [Logrus](https://github.com/sirupsen/logrus)

## Project structure 
- **`cmd/`**  
  Contains the application's main executable logic. This is where `main.go` lives.

- **`internal/`**  
  Contains private application code. Anything inside `internal/` cannot be imported from outside the project (enforced by the Go compiler).

- **`pkg/`**  
  Contains public libraries or utilities that can be imported by other projects if needed.

- **`bin/`**  
  Stores built binaries, generated via the `Makefile`.

- **`Makefile`**  
  Automates build tasks such as compiling, building Docker images, running tests, etc.

- **`go.mod` and `go.sum`**  
  Define module requirements and manage dependencies.

##  Quick Start
### Build the project
```bash
go mod tidy
make build
```

### Run the binary
```bash
./bin/helloworld talk
```

or type:
```bash
go run cmd/main.go talk
```

```console
ERRO[0000] Error detected                                Error="This is an error"
INFO[0000] Talking...                                    Msg="Hello, World!" OtherMsg="Logging is cool!"
Hello, World!
```


### Build and run Docker container
```bash
make container
```

Or without Makefile: 

```bash
docker build -t test/helloworld .
```

```console
Sending build context to Docker daemon  4.503MB
Step 1/4 : FROM alpine
 ---> b0c9d60fc5e3
Step 2/4 : WORKDIR /
 ---> Using cache
 ---> 813578363918
Step 3/4 : COPY ./bin/helloworld /bin
 ---> 8bf1ce271011
Step 4/4 : CMD ["helloworld", "talk"]
 ---> Running in 5dbb96d0225d
Removing intermediate container 5dbb96d0225d
 ---> 0d4933ba1303
Successfully built 0d4933ba1303
Successfully tagged test/helloworld:latest
```

```bash
docker run --rm test/helloworld
```

```console
docker run --rm test/helloworld
Hello, World!
time="2025-04-29T19:15:27Z" level=error msg="Error detected" Error="This is an error"
time="2025-04-29T19:15:27Z" level=info msg=Talking... Msg="Hello, World!" OtherMsg="Logging is cool!"
```

### Running Tests
To run all tests;
```bash
make test 
```

Remember to update Makefile if adding more source directories with tests.

To run all tests in a directory:
```bash
cd pkg/helloworld
go test -v --race
```

Always use the `--race` flag when running tests to detect race conditions during execution.  
The `--race` flag enables the Go race detector, helping you catch concurrency issues early during development.

To run individual test:
```bash
cd pkg/helloworld
go test -v --race -test.run=TestNewHelloWorld
```

```console
=== RUN   TestNewHelloWorld
ERRO[0000] Error detected                                Error="This is an error"
```

## Change the Project Name
To customize the project name, follow these steps:

1. Create a new Git repo.

2. **Edit `go.mod`** 
   Change the module path from: `module github.com/linoss-7/D7024E-Project` to your new project path.

3. **Update Import Paths**  
Modify the import paths in the following files:

- `internal/cli/version.go`  
  Line 6:
  ```go
  "github.com/linoss-7/D7024E-Project/pkg/build"
  ```

- `internal/cli/talk.go`  
  Line 4:
  ```go
  "github.com/linoss-7/D7024E-Project/pkg/helloworld"
  ```

- `cmd/main.go`  
  Lines 4–5:
  ```go
  "github.com/linoss-7/D7024E-Project/internal/cli"
  "github.com/linoss-7/D7024E-Project/pkg/build"
  ```

Replace each instance of `github.com/linoss-7/D7024E-Project` with your new module name.

4. Update Goreleaser
Change the `binary` name to `helloworld` in the `.goreleaser.yml` file.

5. Update Dockerfile 
Change the `helloworld` in the `Dockerfile` file.

6. Update Makefile
Change binary name, and container name:

```console
BINARY_NAME := helloworld
BUILD_IMAGE ?= test/helloworld
PUSH_IMAGE ?= test/helloworld:v1.0.0
```

## Continuous Integration
GitHub will automatically run tests (`make test`) when pushing changes to the `main` branch.

Take a look at these configuration files for CI/CD setup:

- `.github/workflows/docker_master.yaml`
- `.github/workflows/go.yml`
- `.github/workflows/releaser.yaml`
- `.goreleaser.yml`

**Note:**  
The Goreleaser workflow can be used to automatically build and publish binaries on GitHub.  
Click the **Draft a new release** button to create a new release.  
Published releases will appear here: [GitHub Releases - linoss-7/D7024E-Project](https://github.com/linoss-7/D7024E-Project/releases)

The Docker workflow will automatically build and publish a Docker image on GitHub.  
See this page: [GitHub Packages - linoss-7/D7024E-Project](https://github.com/linoss-7/D7024E-Project/pkgs/container/linoss-7/D7024E-Project)

## Other tips
- Run `go mod tidy` to clean up and verify dependencies.
- To store all dependencies in the `./vendor` directory, run:

  ```sh
  go mod vendor
  ```
- Install and use Github Co-pilot! It is very good at generating logging statement.
- Note that build time and current Github take is injected into the binary. Very useful for debugging to know which version you are using. 

```console
./bin/helloworld version                                                                                                                                                              21:53:42
ab76edd
2025-04-29T19:35:04Z
```
