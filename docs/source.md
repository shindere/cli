# Installation from source

0. Verify that you have Go 1.14+ installed

   ```sh
   $ go version
   go version go1.14
   ```

   If `go` is not installed, follow instructions on [the Go website](https://golang.org/doc/install).

1. Clone this repository

   ```sh
   $ git clone https://github.com/cli/cli.git gh-cli
   $ cd gh-cli
   ```

2. Build the project

   ```
   $ make
   ```

3. Create a symbolic link to the `bin/gh` executable in a directory
which appears in your PATH. For example:

   ```sh
   $ sudo ln -s $PWD/bin/gh /usr/local/bin/
   ```

4. Run `gh version` to check if it worked.
