require "formula"

class Algernon < Formula
  homepage "http://algernon.roboticoverlords.org/"
  url "https://github.com/xyproto/algernon/archive/0.71.tar.gz"
  sha1 "8da6f60cbcab9eee68fe9483f23338a0ca1ec196"

  head "https://github.com/xyproto/algernon.git"

  bottle do
    sha1 "1189a998e33951ea6c5a4577b92d5b9090c1d2fe" => :mavericks
    sha1 "765981eef07f07a947cb31a4eaa17962ae945b21" => :mountain_lion
    sha1 "a7f5387d0e3595912dc45bd6f74811b2c4431d45" => :lion
  end

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
