# Fireboom - Developer Onboarding

Welcome to the Fireboom project! This README will guide you through the steps needed to get the project up and running on your local development environment.

### Prerequisites

Before you begin, ensure you have the following installed on your system:

- Git
- Go 1.21 or later

### Cloning the Repository

To get started, clone the Fireboom repository using the following command:

```shell
git clone --recurse-submodules https://github.com/fireboomio/fireboom.git
```

The `--recurse-submodules` flag is essential as it also clones the submodules `fireboom-web` and `wundergraph` that are part of this project.

### Submodules

Fireboom has two submodules:

1. **Frontend assets directory (`assets/front`):**

   This submodule contains all the frontend resources necessary for the project.
   Git repository: https://github.com/fireboomio/fireboom-web.git

2. **Backend resources directory (`wundergraphGitSubmodule`):**

   This submodule contains resources for backend functionality, forked from other open-source projects.
   Git repository: https://github.com/fireboomio/wundergraph.git

To ensure your submodules are up to date, run:

```shell
git submodule update --init --recursive
```

### Building the Project

Fireboom uses Cobra for command-line argument parsing. The first argument is mandatory and can be one of `dev`, `start`, or `build`. Additional optional arguments can be used to override environment variables or toggle other features.

To build the project, navigate to the root directory of the project and run:

```shell
go build -o fireboom
```

### Running the Project

After building, you can run the project using one of the mandatory commands. For example:

```shell
./fireboom dev
```

or

```shell
./fireboom start
```

or

```shell
./fireboom build
```

You can pass additional arguments as needed to customize the environment or functionalities.

### Getting Help

If you need help with the commands or have any issues, you can refer to the Cobra documentation or create an issue in the Fireboom github repository.

### Contributing

We welcome contributions! Please read through our CONTRIBUTING guide to get started on making pull requests.

### License

This project is open-source and is licensed under [LICENSE](LICENSE). Please make sure you comply with the license terms while contributing or using this project.

### Contact

If you have any further questions or require assistance, please open an issue on the Fireboom GitHub repository, and someone from the team will be in touch.

We hope you enjoy working on Fireboom! Your contributions are valuable to us and the open-source community.