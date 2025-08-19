class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "1.5.1"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-darwin-arm64"
      sha256 "ad4e68969827772167e7ce0f7a40ae093ca8ad6baaa4a252b6f17a2cd922764a"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-darwin-amd64"
      sha256 "37cb4e7acd812f02c7e967db35cd3ce9d250c2be140133ac152abe5d870aa442"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/v1.5.1/kportforward-linux-amd64"
    sha256 "ab84094d1d8e0fa14bb7fbfff92866a6a7ab197bba2a4405b18dca459741bac4"
  end

  depends_on "kubectl" => :recommended

  def install
    # Move the downloaded binary to the bin directory with the name "kportforward"
    # First, find what files we have in the current directory
    binary = Dir["*"].first
    bin.install binary => "kportforward"
    
    # Ensure binary is executable
    chmod 0755, bin/"kportforward"
  end

  test do
    assert_match(/kportforward/i, shell_output("#{bin}/kportforward version 2>&1", 2))
  end
end
