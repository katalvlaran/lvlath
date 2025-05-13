# Contributing to lvlath

Welcome! 🎉 We're thrilled you're considering contributing to **lvlath**, our in‑memory, thread‑safe Go graph library. Whether you're filing an issue, improving docs, or submitting your first PR—thank you!

---

## 📝 Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). We aim to foster a welcoming, respectful community.

---

## 🚀 Getting Started

1. **Fork** the repo & `git clone` your fork:

   ```sh
   git clone https://github.com/katalvlaran/lvlath.git
   cd lvlath
   ```
2. **Create** a feature branch:

   ```sh
   git checkout -b feature/your-feature-name
   ```
3. **Install** dependencies and ensure Go 1.21+

   ```sh
   go mod tidy
   ```
4. **Run tests & lint** locally:

   ```sh
   go test ./...  
   golangci-lint run
   ```

---

## 🐛 Reporting Issues

* Search existing issues: maybe someone else reported it already! 🔍
* Create a **clear** issue with:

    * Title that summarizes the problem
    * Steps to reproduce
    * Expected vs. actual behavior
    * Go version & OS

---

## 💡 Submitting Pull Requests

1. **Keep PRs small** and focused on a single change.
2. **Update** or **add** tests for any new functionality.
3. **Document** public API changes with GoDoc comments.
4. **Reference** related issues by number (e.g. `Fixes #123`).
5. **Run CI** and ensure all checks pass before marking **Ready for review**.

---

## 🌿 Branching & GitFlow

We follow a lightweight GitFlow:

* **`main`**: always stable, reflects the latest released version.
* **`v0.1.0-alpha`**: development branch for the upcoming alpha release.
* **Feature branches**: `feature/<name>` off of the development branch.
* **Hotfix branches**: `hotfix/<name>` off of `main` for urgent fixes.

Merge into the development branch via pull request; CI must pass.

---

## 🔧 Testing & Benchmarks

* All new code **must** include unit tests (use testify/suite).
* For performance‑critical algorithms, include benchmarks:

  ```go
  func BenchmarkDinic(b *testing.B) { ... }
  ```
* Aim for **>90%** coverage for core and critical algorithms.

---

## 🎨 Style & Linting

* Run `go fmt ./...` and `go vet ./...` before committing.
* Follow [golangci-lint recommended rules](https://golangci-lint.run/).
* Keep function and variable names **clear** and **consistent**.

---

## 🤖 CI Requirements

Our GitHub Actions pipeline (`.github/workflows/go.yml`) enforces:

* `go test ./... -coverprofile=coverage.out`
* `golangci-lint run`
* Codecov upload

All checks must pass for merges.

---

## 📬 Staying in Touch

* Join discussions in **Issues** and **Discussions**.
* Ping maintainers via GitHub if you need help.

Thank you for helping make lvlath better! 🌟
