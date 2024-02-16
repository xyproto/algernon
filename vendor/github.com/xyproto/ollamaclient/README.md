# ollamaclient

This is a Go package for using Ollama.

The default model is `nous-hermes:7b-llama2-q2_K`.

### Getting started

1. Install `ollama` and start it as a service.
2. Run `ollama pull `nous-hermes:7b-llama2-q2_K` to fetch the `nous-hermes:7b-llama2-q2_K` model.
3. Install the summarizer utility: `go install github.com/xyproto/ollamaclient/cmd/summarize@latest`
4. Summarize a README.md file and a source code file: `summarize README.md ollamaclient.go`
5. Write a poem about one or more files: `summarize --prompt "Write a poem about the following files:" README.md`

### Usage of the `summarize` utility

```bash
./summarize [flags] <filename1> [<filename2> ...]
```

#### Flags

- `-m`, `--model`: Specify an Ollama model. The default is `nous-hermes:latest`.
- `-o`, `--output`: Define an output file to store the summary.
- `-p`, `--prompt`: Specify a custom prompt header for summary. The default is `Write a short summary of a project that contains the following files:`
- `-w`, `--wrap`: Set the word wrap width. Use -1 to detect the terminal width.
- `-v`, `--version`: Display the current version.
- `-V`, `--verbose`: Enable verbose logging.

#### Example use

Generate a summary with a custom prompt:

```bash
./summarize -w -1 -p "Summarize these files:" README.md CONFIG.md
```

Generate a summary, saving the output to a file:

```bash
./summarize -o output.txt README.md CONFIG.md
```

Generate a summary with custom word wrap width:

```bash
./summarize -w 100 README.md
```

### Environment variables

These environment variables are supported:

* `OLLAMA_HOST` (`http://localhost:11434` by default)
* `OLLAMA_MODEL` (`nous-hermes:7b-llama2-q2_K` by default
* `OLLAMA_VERBOSE` (`false` by default)

### General info

* Version: 1.5.0
* License: Apache2
* Author: Alexander F. Rødseth
