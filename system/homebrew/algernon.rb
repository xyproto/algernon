class Algernon < Formula
  desc "HTTP/2 web server with Lua support"
  homepage "http://algernon.roboticoverlords.org/"
  url "https://github.com/xyproto/algernon/archive/0.74.tar.gz"
  sha256 "1341af6864643a968d85bfa63ca231604b6d1123919c6826ae179908c6c4a176"
  head "https://github.com/xyproto/algernon.git"

  depends_on "go" => :build
  depends_on :hg => :build
  depends_on "readline"

  def install
    ENV["GOPATH"] = buildpath

    # Fetch Go dependencies
    system "go", "get", "-d"

    # Build and install the executable
    system "go", "build", "-o", "algernon"
    bin.install "algernon"
  end

  test do
    begin
      # Start serving HTTP on port 3000
      fork_pid = fork do
        spawn("#{bin}/algernon", "--httponly", "--server", "--addr", ":45678",
              "--boltdb", "/tmp/_brew_test.db", :out=>"/dev/null", :err=>"/dev/null")
      end
      child_pid = fork_pid + 1
      # Detach the fork
      Process.detach fork_pid

      # Wait for the server to start serving
      sleep(0.5)

      # Check that we have the right PID
      pgrep_output = `pgrep algernon`
      assert_equal 1, pgrep_output.count("\n")
      assert_equal pgrep_output.to_i, child_pid
      algernon_pid = child_pid

      # Check that the server is responding correctly
      output = `curl -sIm3 -o- http://localhost:45678`
      assert output.include?("Server: Algernon")
      assert_equal 0, $?.exitstatus
    ensure
      # Stop the server gracefully
      Process.kill("HUP", algernon_pid)

      # Remove temporary Bolt database
      `rm -f /tmp/_brew_test.db`
    end
  end
end
