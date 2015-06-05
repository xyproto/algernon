class Algernon < Formula
  desc "HTTP/2 web server with Lua support"
  homepage "http://algernon.roboticoverlords.org/"
  url "https://github.com/xyproto/algernon/archive/0.74.tar.gz"
  version "0.74"
  sha256 "eb6cbdad6c497f06732badaf837a01dc13272dd31420c688a0f297136553db5e"

  head "https://github.com/xyproto/algernon.git"

  depends_on "go" => :build
  depends_on :hg => :build

  def install
    ENV["GOPATH"] = buildpath

    # Install Go dependencies
    system "go", "get", "-d"

    # Build and install algernon
    system "go", "build", "-o", "algernon"

    bin.install "algernon"
  end

  test do
    system "#{bin}/algernon", "--version"
  end
end
