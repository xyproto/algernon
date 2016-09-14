# Contributing

To hack on this project:

1. Install as usual (`go get -u github.com/...`)
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Ensure everything works and the tests pass (`go test`)
4. Commit your changes (`git commit -am 'Add some feature'`)

Contribute upstream:

1. Fork it on GitHub
2. Add your remote (`git remote add fork git@github.com:mycompany/repo.git`)
3. Push to the branch (`git push fork my-new-feature`)
4. Create a new Pull Request on GitHub

For other team members:

1. Install as usual (`go get -u github.com/...`)
2. Add your remote (`git remote add fork git@github.com:mycompany/repo.git`)
3. Pull your revisions (`git fetch fork; git checkout -b my-new-feature fork/my-new-feature`)

Notice: Always use the original import path by installing with `go get`.
