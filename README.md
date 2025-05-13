
# docbot

**docbot** is a modular document bot for interacting with and indexing Google Docs. It provides both a command-line interface and a lightweight web interface. The architecture is modular, separating concerns into packages for Google API access, document processing, transaction handling, and user interaction.

---

## Features

- Index and search Google Docs content
- Serve documents and search via a web interface
- Use the command-line interface for doc listing and access
- Integrates with Google APIs for Docs, Comments, and Permissions
- Modular codebase designed for extensibility
- Includes sample templates and test datasets
- Docker-compatible with `supervisord` support

---

## Project Structure

```
docbot/
├── main.go                  # Entry point
├── bot/                     # Bot core: document indexing and search logic
├── cli/                     # CLI tooling and output templates
├── google/                  # Google Docs/Drive API access
├── transaction/             # Document transactions and session utilities
├── web/                     # Web frontend and templates
├── util/                    # General utilities
├── Dockerfile               # Container definition
├── Makefile                 # Build and dev tasks
├── supervisord.conf         # Supervisor process configuration
```

---

## Installation

### Prerequisites

- Go 1.18 or newer
- Google Cloud project with OAuth2 credentials
- Docker (optional, for containerized runs)

### Clone the repo

```bash
git clone https://github.com/ciwg/docbot.git
cd docbot
```

### Build the binary

```bash
go build -o docbot main.go
```

---

## Usage

### Start the Web Server

```bash
go run main.go
```

Then visit:  
`http://localhost:8080`

### Use the CLI

Example to list documents:

```bash
go run cli/cli.go ls
```

Templates for CLI output are located under `cli/template/`.

---

## Google API Setup

1. Create a project in Google Cloud Console.
2. Enable the **Google Docs API** and **Google Drive API**.
3. Download the `credentials.json` file and place it in the project root.
4. On first run, authenticate via the browser when prompted.

---

## Testing

Run the tests for all modules:

```bash
go test ./...
```

Each major package includes unit tests and test data in subfolders like `testdata/`.

---

## License

This project is licensed under the **BSD 3-Clause License**.

```
Copyright (c) 2022, stevegt
All rights reserved.
```

See the [LICENSE](LICENSE) file for full license text.

---

## Acknowledgments

Created and maintained by **stevegt**. Contributions welcome via pull requests or issue discussions.
