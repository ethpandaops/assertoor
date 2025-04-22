# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/noku-team/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.24` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/noku-team/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
