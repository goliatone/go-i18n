# Examples

The `examples` directory hosts runnable snippets that demonstrate how the `i18n` toolkit can be wired inside an application.

## Available scenarios

- `basic`: loads file-based translations, builds a translator through `Config`, wires template helpers, and shows how to override formatters.

## Running the demo

Use the provided command entrypoint:

```bash
go run ./cmd/example
```

This renders a sample invoice in Spanish, prints translation hook output, and showcases fallback + missing key handling.
