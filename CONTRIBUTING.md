# Contributing to tuikit

Thanks for your interest in tuikit.

## Before you start

Open an issue to discuss a change before sending a pull request, especially a
new widget, dialog or exported API. tuikit is a library with users outside
goSSMS, so exported surface is not removed or narrowed lightly.

Read these first:

- `README.md` § Design principles, § Adding a new control, § Adding a new
  dialog in an application;
- `ARCHITECTURE.md` § Mouse gestures before touching any `HandleMouse`.

## Rules that shape the code

- Dependencies are the standard library, tcell v3 and displaywidth, nothing
  else. `TestTuikitImportsOnlyItsDeclaredDependencies` and `TestTuikitLayering`
  enforce this and the one-way package graph.
- Use `core.DisplayWidth`, never `len(s)`, for column math.
- Check `go doc` or the module cache before writing a tcell call; tcell v3 has
  no `PollEvent`/`PostEvent` (use the `EventQ()` channel).
- Match the surrounding comment density. Long "why" comments record shipped
  bugs and are not a cleanup target.

## Checks

```sh
go build ./... && go vet ./... && gofmt -l .
go test ./...
staticcheck ./...
go test -race -count=1 ./<pkg>/...   # a change crossing goroutines
```

`gofmt -l .` must print nothing and `staticcheck` must stay clean. Passing
tests are not enough for UI changes: drive the widget in a real terminal (for
example through goSSMS) before calling it done.

## Pull requests

- One logical change per PR, with tests for new behaviour.
- Explain what changed and why; link the issue.
- Do not edit generated or release files (version tags are the maintainer's).

## Security

Do not report vulnerabilities in a public issue. Use GitHub's private
vulnerability reporting on the repository's Security tab.

## Licence

By contributing you agree that your contribution is licensed under the MIT
licence (see `LICENSE`).
