# Contributing to devctl

Thanks for helping to make devctl better!

## Setup

**Requirements**:
- Go 1.26+
- Git 2.0+
- Bash

**Clone the repository**:
```bash
git clone https://github.com/rstanziale/devctl
cd devctl
go mod download
go build -o devctl main.go
```

## Branching

Use descriptive branch names with prefixes (from `main`):

- `feature/description` - new features
- `hotfix/description` - bug fixes
- `refactor/description` - refactoring
- `docs/description` - documentation
- `test/description` - tests
- `chore/description` - maintenance

Example:

```bash
git checkout main
git pull
git checkout -b feature/add-new-command
```

## Making a PR

- Check your changes locally:
    ```bash
    go test ./...
    ```

- Push your branch:
    ```bash
    git push origin feature/add-new-command
    ```

- Open a PR [here](https://github.com/rstanziale/devctl/compare) and fill out the template
- Wait for maintainers to review and provide feedback

## Code Guidelines

- **Tests**: Add `*_test.go` files for new features
- **Errors**: Always handle errors explicitly
- **Comments**: Write clear, concise comments in English
- **Functions**: Keep them small and focused

---

Questions? [Open an issue](https://github.com/rstanziale/devctl/issues) or reach out to maintainers.
