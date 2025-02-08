--- run the printf command that ouytputs three lines: line1, line2 and line3
stdout, stderr, exitcode = run3("printf 'line1\nline2\nline3\n'")

--- pretty print the stdout table
print("--- stdout ---")
pprint(stdout)

--- pretty print the stderr table
print("--- stderr ---")
pprint(stderr)

--- print the exit code
print("--- exit code ---")
print(code)
